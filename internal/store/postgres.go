package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/randy-girard/flynn-discovery/migrations"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func OpenPostgres(databaseURL string) (*Postgres, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	p := &Postgres{pool: pool}
	if err := p.migrate(); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

func (p *Postgres) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

func (p *Postgres) migrate() error {
	ctx := context.Background()
	if _, err := p.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_version (version text PRIMARY KEY)`); err != nil {
		return fmt.Errorf("schema_version: %w", err)
	}
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		var applied string
		err := p.pool.QueryRow(ctx, `SELECT version FROM schema_version WHERE version = $1`, name).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		body, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_version (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgres) CreateCluster(c *Cluster) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO clusters (cluster_id, creator_ip, creator_user_agent) VALUES ($1, $2, $3) RETURNING created_at`,
		c.ID, c.CreatorIP, c.CreatorUserAgent,
	).Scan(&c.CreatedAt)
}

func (p *Postgres) DefaultCluster() (*Cluster, error) {
	c := &Cluster{}
	err := p.pool.QueryRow(context.Background(),
		`SELECT cluster_id, creator_ip, creator_user_agent, created_at FROM clusters ORDER BY created_at ASC, cluster_id ASC LIMIT 1`,
	).Scan(&c.ID, &c.CreatorIP, &c.CreatorUserAgent, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (p *Postgres) CreateInstance(inst *Instance) error {
	if inst.ID == "" {
		inst.ID = uuid.NewString()
	}
	if inst.SSHPublicKeys == nil {
		inst.SSHPublicKeys = []SSHPublicKey{}
	}
	keys, err := json.Marshal(inst.SSHPublicKeys)
	if err != nil {
		return err
	}
	var created time.Time
	err = p.pool.QueryRow(context.Background(),
		`INSERT INTO instances (instance_id, cluster_id, flynn_version, ssh_public_keys, url, name, creator_ip)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`,
		inst.ID, inst.ClusterID, inst.FlynnVersion, keys, inst.URL, inst.Name, inst.CreatorIP,
	).Scan(&created)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if err := p.loadInstanceByURL(inst); err != nil {
				return err
			}
			return ErrExists
		}
		return err
	}
	inst.CreatedAt = &created
	return nil
}

func (p *Postgres) loadInstanceByURL(inst *Instance) error {
	var keys []byte
	var created time.Time
	err := p.pool.QueryRow(context.Background(),
		`SELECT instance_id, flynn_version, ssh_public_keys, url, name, creator_ip, created_at
		 FROM instances WHERE cluster_id = $1 AND url = $2`,
		inst.ClusterID, inst.URL,
	).Scan(&inst.ID, &inst.FlynnVersion, &keys, &inst.URL, &inst.Name, &inst.CreatorIP, &created)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(keys, &inst.SSHPublicKeys); err != nil {
		return err
	}
	inst.CreatedAt = &created
	return nil
}

func (p *Postgres) GetClusterInstances(clusterID string) ([]*Instance, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT instance_id, flynn_version, ssh_public_keys, url, name, creator_ip, created_at
		 FROM instances WHERE cluster_id = $1 ORDER BY created_at ASC`,
		clusterID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Instance
	for rows.Next() {
		inst := &Instance{ClusterID: clusterID}
		var keys []byte
		var created time.Time
		if err := rows.Scan(&inst.ID, &inst.FlynnVersion, &keys, &inst.URL, &inst.Name, &inst.CreatorIP, &created); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(keys, &inst.SSHPublicKeys); err != nil {
			return nil, err
		}
		createdCopy := created
		inst.CreatedAt = &createdCopy
		out = append(out, inst)
	}
	return out, rows.Err()
}
