module github.com/randy-girard/flynn-plugin-discovery

go 1.24.0

toolchain go1.24.12

require (
	github.com/flynn/flynn v0.0.0-20260914140432-be1b4a311248
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.8.0
)

require (
	github.com/docker/go-units v0.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jtacoma/uritemplates v1.0.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/opencontainers/runc v1.0.0-rc8 // indirect
	github.com/opencontainers/runtime-spec v1.0.1 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace github.com/flynn/flynn => github.com/randy-girard/flynn v0.0.0-20260914140432-be1b4a311248
