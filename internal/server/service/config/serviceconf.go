// Package config provides configuration for the service layer.
package config

// ServiceConf holds the master encryption key for wrapping user keys.
type ServiceConf struct {
	MasterKey string `json:"master_key"`
}
