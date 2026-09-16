package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Memory struct {
	mu        sync.Mutex
	clusters  map[string]*Cluster
	instances map[string][]*Instance
	byURL     map[string]*Instance
}

func NewMemory() *Memory {
	return &Memory{
		clusters:  map[string]*Cluster{},
		instances: map[string][]*Instance{},
		byURL:     map[string]*Instance{},
	}
}

func (m *Memory) CreateCluster(c *Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	cp := *c
	m.clusters[c.ID] = &cp
	return nil
}

func (m *Memory) DefaultCluster() (*Cluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.clusters) == 0 {
		return nil, nil
	}
	all := make([]*Cluster, 0, len(m.clusters))
	for _, c := range m.clusters {
		cp := *c
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID < all[j].ID
		}
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	return all[0], nil
}

func (m *Memory) CreateInstance(inst *Instance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.clusters[inst.ClusterID]; !ok {
		return errors.New("unknown cluster")
	}
	key := inst.ClusterID + "\x00" + inst.URL
	if existing, ok := m.byURL[key]; ok {
		*inst = *existing
		return ErrExists
	}
	if inst.ID == "" {
		inst.ID = uuid.NewString()
	}
	if inst.SSHPublicKeys == nil {
		inst.SSHPublicKeys = []SSHPublicKey{}
	}
	now := time.Now().UTC()
	inst.CreatedAt = &now
	cp := *inst
	cp.SSHPublicKeys = append([]SSHPublicKey{}, inst.SSHPublicKeys...)
	m.instances[inst.ClusterID] = append(m.instances[inst.ClusterID], &cp)
	m.byURL[key] = &cp
	return nil
}

func (m *Memory) GetClusterInstances(clusterID string) ([]*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := m.instances[clusterID]
	out := make([]*Instance, 0, len(src))
	for _, inst := range src {
		cp := *inst
		cp.SSHPublicKeys = append([]SSHPublicKey{}, inst.SSHPublicKeys...)
		out = append(out, &cp)
	}
	return out, nil
}
