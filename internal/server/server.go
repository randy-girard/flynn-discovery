package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/randy-girard/flynn-plugin-discovery/internal/store"
)

type Server struct {
	URL     string
	Backend store.Backend
	mux     *http.ServeMux
}

func New(url string, backend store.Backend) *Server {
	s := &Server{URL: strings.TrimRight(url, "/"), Backend: backend, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /.well-known/status", s.status)
	s.mux.HandleFunc("GET /.well-known/cluster", s.wellKnownCluster)
	s.mux.HandleFunc("POST /clusters", s.createCluster)
	s.mux.HandleFunc("POST /clusters/{cluster_id}/instances", s.createInstance)
	s.mux.HandleFunc("GET /clusters/{cluster_id}/instances", s.getInstances)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ClusterURL(id string) string {
	if s.URL != "" {
		return s.URL + "/clusters/" + id
	}
	return "/clusters/" + id
}

func (s *Server) EnsureDefaultCluster() (*store.Cluster, error) {
	c, err := s.Backend.DefaultCluster()
	if err != nil {
		return nil, err
	}
	if c != nil {
		return c, nil
	}
	c = &store.Cluster{CreatorUserAgent: "flynn-plugin-discovery"}
	if err := s.Backend.CreateCluster(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"healthy"}`)
}

func (s *Server) wellKnownCluster(w http.ResponseWriter, r *http.Request) {
	c, err := s.EnsureDefaultCluster()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"id":  c.ID,
			"url": s.ClusterURL(c.ID),
		},
	})
}

func (s *Server) createCluster(w http.ResponseWriter, r *http.Request) {
	// This plugin serves one Flynn cluster. Reuse the default cluster so
	// `flynn-host init --init-discovery` against this URL joins the same
	// token instead of minting an empty one.
	if existing, err := s.Backend.DefaultCluster(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if existing != nil {
		w.Header().Set("Location", s.ClusterURL(existing.ID))
		w.WriteHeader(http.StatusCreated)
		return
	}
	c := &store.Cluster{
		CreatorIP:        sourceIP(r),
		CreatorUserAgent: r.Header.Get("User-Agent"),
	}
	if len(c.CreatorUserAgent) > 1000 {
		c.CreatorUserAgent = c.CreatorUserAgent[:1000]
	}
	if err := s.Backend.CreateCluster(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", s.ClusterURL(c.ID))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) createInstance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Data *store.Instance `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Data == nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	inst := body.Data
	inst.ClusterID = r.PathValue("cluster_id")
	inst.CreatorIP = sourceIP(r)
	status := http.StatusCreated
	if err := s.Backend.CreateInstance(inst); err == store.ErrExists {
		status = http.StatusConflict
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("%s/instances/%s", s.ClusterURL(inst.ClusterID), inst.ID))
	writeJSON(w, status, map[string]any{"data": inst})
}

func (s *Server) getInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := s.Backend.GetClusterInstances(r.PathValue("cluster_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if instances == nil {
		instances = []*store.Instance{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": instances})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func sourceIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[len(ips)-1])
	}
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return ip
}
