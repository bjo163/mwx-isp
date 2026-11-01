package config

import (
	"path/filepath"
	"time"
)

// DefaultWebSecret is the built-in development placeholder.
const DefaultWebSecret = "dev-secret-please-change-in-production"

// DBConfig holds database configuration.
type DBConfig struct {
	Type            string
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	Passwd          string // alias for Password
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	Datasource      string
	Debug           bool
	IdleConn        int
	MaxConn         int
}

// SysConfig holds system-level configuration.
type SysConfig struct {
	Appid    string
	Location string
	Workdir  string
	Debug    bool
}

// LoggerConfig holds logger configuration.
type LoggerConfig struct {
	Mode        string
	FileEnable  bool
	Filename    string
}

// WebConfig holds web server configuration.
type WebConfig struct {
	Secret     string
	Host       string
	Port       int
	TlsEnabled bool
	TlsPort    int
}

// TenantConfig holds tenant limits.
type TenantConfig struct {
	MaxTenantCount    int
	MaxOperatorCount  int
	MaxNasCount       int
	MaxOnlineCount    int
	MaxSimultaneous   int
	MaxTotalOnline    int
}

// BackupConfig holds backup configuration.
type BackupConfig struct {
	Passphrase string
	Dir        string
}

// RadiusConfig holds RADIUS server configuration.
type RadiusConfig struct {
	BindAddress  string
	AuthPort     int
	AcctPort     int
	MaxSessions  int
	Enabled      bool
	Host         string
	Debug        bool
	RadsecPort   int
	RadsecWorker int
}

// AppConfig represents the application configuration structure.
type AppConfig struct {
	Database  DBConfig
	System    SysConfig
	Logger    LoggerConfig
	Web       WebConfig
	Tenant    TenantConfig
	SysConfig string
	Radiusd   RadiusConfig
	Backup    BackupConfig
}

// LoadConfig loads configuration from a YAML file.
// Note: This is a simplified implementation. In production, you would
// use a proper YAML parser like gopkg.in/yaml.v3.
func LoadConfig(filename string) *AppConfig {
	// For now, return default values. In a real implementation,
	// this would parse the YAML file and populate the struct.
	return &AppConfig{
		Database: DBConfig{
			Type:            "sqlite",
			Name:            "mwx-isp.db",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
		},
		System: SysConfig{
			Appid:    "mwx-isp",
			Location: "UTC",
			Workdir:  "./data",
			Debug:    true,
		},
		Logger: LoggerConfig{
			Mode:        "development",
			FileEnable:  false,
			Filename:    "mwx-isp.log",
		},
		Web: WebConfig{
			Secret:     DefaultWebSecret,
			Host:       "0.0.0.0",
			Port:       3000,
			TlsEnabled: false,
			TlsPort:    3001,
		},
		Tenant: TenantConfig{
			MaxTenantCount:    1000,
			MaxOperatorCount:  10000,
			MaxNasCount:       10000,
			MaxOnlineCount:    100000,
			MaxSimultaneous:   1000,
			MaxTotalOnline:    1000000,
		},
		SysConfig: "./data",
		Radiusd: RadiusConfig{
			BindAddress:  "0.0.0.0",
			AuthPort:     1812,
			AcctPort:     1813,
			MaxSessions:  10000,
			Enabled:      true,
			Host:         "0.0.0.0",
			Debug:        true,
			RadsecPort:   2083,
			RadsecWorker: 4,
		},
		Backup: BackupConfig{
			Passphrase: "",
			Dir:        "",
		},
	}
}

// GetDataDir returns the data directory path.
func (c *AppConfig) GetDataDir() string {
	if c.SysConfig != "" {
		return c.SysConfig
	}
	return c.System.Workdir
}

// GetPrivateDir returns the private directory path.
func (c *AppConfig) GetPrivateDir() string {
	if c.SysConfig != "" {
		return c.SysConfig
	}
	return c.System.Workdir
}

// GetBackupDir returns the backup directory path.
func (c *AppConfig) GetBackupDir() string {
	if c.Backup.Dir != "" {
		return c.Backup.Dir
	}
	return filepath.Join(c.GetPrivateDir(), "backups")
}

// GetRadsecCaCertPath returns the RADSEC CA certificate path.
func (c *AppConfig) GetRadsecCaCertPath() string {
	return filepath.Join(c.GetPrivateDir(), "radsec-ca.crt")
}

// GetRadsecCertPath returns the RADSEC server certificate path.
func (c *AppConfig) GetRadsecCertPath() string {
	return filepath.Join(c.GetPrivateDir(), "radsec-server.crt")
}

// GetRadsecKeyPath returns the RADSEC server key path.
func (c *AppConfig) GetRadsecKeyPath() string {
	return filepath.Join(c.GetPrivateDir(), "radsec-server.key")
}