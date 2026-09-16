package store

import (
	"errors"
	"time"
)

var ErrExists = errors.New("object exists")

type Cluster struct {
	ID               string
	CreatorIP        string
	CreatorUserAgent string
	CreatedAt        time.Time
}

type Instance struct {
	ID            string         `json:"id"`
	ClusterID     string         `json:"cluster_id"`
	FlynnVersion  string         `json:"flynn_version,omitempty"`
	SSHPublicKeys []SSHPublicKey `json:"ssh_public_keys,omitempty"`
	URL           string         `json:"url,omitempty"`
	Name          string         `json:"name,omitempty"`
	CreatorIP     string         `json:"-"`
	CreatedAt     *time.Time     `json:"created_at,omitempty"`
}

type SSHPublicKey struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

// Backend is the discovery API store. Memory is used in tests; Postgres is
// used when the plugin is installed (DATABASE_URL).
type Backend interface {
	CreateCluster(*Cluster) error
	DefaultCluster() (*Cluster, error)
	CreateInstance(*Instance) error
	GetClusterInstances(clusterID string) ([]*Instance, error)
}
