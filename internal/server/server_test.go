package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/randy-girard/flynn-plugin-discovery/internal/store"
)

func TestClusterURL(t *testing.T) {
	s := New("https://discovery.example.com/", store.NewMemory())
	if got := s.ClusterURL("abc"); got != "https://discovery.example.com/clusters/abc" {
		t.Fatal(got)
	}
	s.URL = ""
	if got := s.ClusterURL("abc"); got != "/clusters/abc" {
		t.Fatal(got)
	}
}

func TestDiscoveryAPI(t *testing.T) {
	s := New("https://discovery.example.com", store.NewMemory())
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/.well-known/status")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "healthy") {
		t.Fatalf("status %d %s", res.StatusCode, body)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "flynn-host/test")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create cluster %d", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://discovery.example.com/clusters/") {
		t.Fatalf("Location=%s", loc)
	}
	clusterID := strings.TrimPrefix(loc, "https://discovery.example.com/clusters/")
	instURL := ts.URL + "/clusters/" + clusterID + "/instances"

	payload := `{"data":{"name":"node1","url":"http://10.0.0.1:1113"}}`
	res, err = http.Post(instURL, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		Data store.Instance `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusCreated || created.Data.ID == "" || created.Data.Name != "node1" {
		t.Fatalf("create instance %d %+v", res.StatusCode, created.Data)
	}

	res, err = http.Post(instURL, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("dup status %d", res.StatusCode)
	}

	res, err = http.Get(ts.URL + "/clusters/" + clusterID + "/instances")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Data []*store.Instance `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || len(list.Data) != 1 || list.Data[0].URL != "http://10.0.0.1:1113" {
		t.Fatalf("list %+v status %d", list.Data, res.StatusCode)
	}

	res, err = http.Get(ts.URL + "/.well-known/cluster")
	if err != nil {
		t.Fatal(err)
	}
	var wellKnown struct {
		Data struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&wellKnown); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if wellKnown.Data.ID != clusterID || wellKnown.Data.URL != loc {
		t.Fatalf("well-known %+v want %s %s", wellKnown.Data, clusterID, loc)
	}

	req2, err := http.NewRequest(http.MethodPost, ts.URL+"/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err = http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusCreated || res.Header.Get("Location") != loc {
		t.Fatalf("second POST must reuse default cluster, status=%d loc=%s", res.StatusCode, res.Header.Get("Location"))
	}
}

func TestEnsureDefaultCluster(t *testing.T) {
	s := New("https://discovery.example.com", store.NewMemory())
	first, err := s.EnsureDefaultCluster()
	if err != nil || first.ID == "" {
		t.Fatalf("%+v %v", first, err)
	}
	second, err := s.EnsureDefaultCluster()
	if err != nil || second.ID != first.ID {
		t.Fatalf("must reuse %s vs %s", first.ID, second.ID)
	}
}

func TestCreateInstanceRejectsInvalidJSON(t *testing.T) {
	s := New("", store.NewMemory())
	ts := httptest.NewServer(s)
	defer ts.Close()
	res, err := http.Post(ts.URL+"/clusters/x/instances", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}
