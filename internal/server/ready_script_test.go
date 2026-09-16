package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/randy-girard/flynn-discovery/internal/store"
)

func TestReadyScriptRegistersHost(t *testing.T) {
	backend := store.NewMemory()
	s := New("", backend)
	if _, err := s.EnsureDefaultCluster(); err != nil {
		t.Fatal(err)
	}
	discovery := httptest.NewServer(s)
	defer discovery.Close()
	s.URL = strings.TrimRight(discovery.URL, "/")

	hostStatus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":  "node1",
			"url": "http://10.0.0.1:1113",
		})
	}))
	defer hostStatus.Close()

	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "discovery-token")
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	script := filepath.Join(filepath.Dir(thisFile), "..", "..", "script", "ready.sh")

	run := func() []byte {
		cmd := exec.Command("bash", script)
		cmd.Env = append(os.Environ(),
			"FLYNN_PLUGIN_NAME=discovery",
			"DISCOVERY_INTERNAL_URL="+discovery.URL,
			"URL="+s.URL,
			"DISCOVERY_TOKEN_FILE="+tokenFile,
			"FLYNN_HOST_STATUS_URL="+hostStatus.URL,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return out
	}

	out := run()
	got, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimSpace(string(got))
	if !strings.HasPrefix(token, s.URL+"/clusters/") {
		t.Fatalf("token=%q\n%s", token, out)
	}
	if !strings.Contains(string(out), "sudo flynn-host init --discovery "+token) {
		t.Fatalf("missing join command\n%s", out)
	}
	c, err := backend.DefaultCluster()
	if err != nil {
		t.Fatal(err)
	}
	list, err := backend.GetClusterInstances(c.ID)
	if err != nil || len(list) != 1 || list[0].URL != "http://10.0.0.1:1113" {
		t.Fatalf("instances=%+v err=%v", list, err)
	}

	run()
	list, err = backend.GetClusterInstances(c.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("dup register instances=%+v err=%v", list, err)
	}
}
