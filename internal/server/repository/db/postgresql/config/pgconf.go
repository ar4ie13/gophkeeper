package config

// PGConf contains Postgres configuration
type PGConf struct {
	DatabaseDSN string `json:"database_dsn,omitempty"`
}
