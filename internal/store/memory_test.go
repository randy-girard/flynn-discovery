package store

import (
	"testing"
	"time"
)

func TestMemoryCreateClusterAndInstances(t *testing.T) {
	m := NewMemory()
	c, err := m.DefaultCluster()
	if err != nil || c != nil {
		t.Fatalf("empty default: %v %v", c, err)
	}
	cluster := &Cluster{CreatorIP: "1.2.3.4", CreatorUserAgent: "flynn-host/dev"}
	if err := m.CreateCluster(cluster); err != nil || cluster.ID == "" {
		t.Fatalf("create cluster: %v %+v", err, cluster)
	}
	got, err := m.DefaultCluster()
	if err != nil || got == nil || got.ID != cluster.ID {
		t.Fatalf("default: %+v %v", got, err)
	}

	inst := &Instance{ClusterID: cluster.ID, Name: "node1", URL: "http://10.0.0.1:1113"}
	if err := m.CreateInstance(inst); err != nil || inst.ID == "" || inst.CreatedAt == nil {
		t.Fatalf("create instance: %v %+v", err, inst)
	}
	dup := &Instance{ClusterID: cluster.ID, Name: "node1-again", URL: "http://10.0.0.1:1113"}
	if err := m.CreateInstance(dup); err != ErrExists {
		t.Fatalf("dup err=%v", err)
	}
	if dup.ID != inst.ID {
		t.Fatalf("dup should return existing id %s got %s", inst.ID, dup.ID)
	}

	second := &Instance{ClusterID: cluster.ID, Name: "node2", URL: "http://10.0.0.2:1113"}
	if err := m.CreateInstance(second); err != nil {
		t.Fatal(err)
	}
	list, err := m.GetClusterInstances(cluster.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	if err := m.CreateInstance(&Instance{ClusterID: "missing", URL: "http://x", Name: "x"}); err == nil {
		t.Fatal("unknown cluster")
	}
}

func TestMemoryDefaultClusterOldest(t *testing.T) {
	m := NewMemory()
	first := &Cluster{ID: "aaaa", CreatedAt: time.Unix(1, 0).UTC()}
	second := &Cluster{ID: "bbbb", CreatedAt: time.Unix(2, 0).UTC()}
	if err := m.CreateCluster(second); err != nil {
		t.Fatal(err)
	}
	if err := m.CreateCluster(first); err != nil {
		t.Fatal(err)
	}
	got, err := m.DefaultCluster()
	if err != nil || got.ID != "aaaa" {
		t.Fatalf("%+v %v", got, err)
	}
}
