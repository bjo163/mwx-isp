package app

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/config"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/config"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// getDatabase opens exactly the backend selected by configuration. A failed
// PostgreSQL connection is a startup error: silently switching to a fresh local
// SQLite file would split production writes across unrelated databases.
func getDatabase(dbConfig config.DBConfig, workdir string) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(dbConfig.Type)) {
	case "sqlite":
		return getSqliteDatabase(dbConfig, workdir), nil
	case "postgres", "postgresql":
		return getPgDatabase(dbConfig)
	default:
		return nil, fmt.Errorf("unsupported database type %q (supported: postgres, sqlite)", dbConfig.Type)
	}
}

// getSqliteDatabase returns a SQLite database connection
func getSqliteDatabase(config config.DBConfig, workdir string) *gorm.DB {
	// e.g., if the name is not an absolute path and not an in-memory DB, store it under workdir/data
	dbPath := config.Name
	if dbPath == "" {
		dbPath = "mwx-isp.db"
	}
	if dbPath != ":memory:" {
		if !filepath.IsAbs(dbPath) {
			// If user specified just filename or relative path, prefix with workdir/data
			// unless it already starts with data or workdir
			cleaned := filepath.Clean(dbPath)
			if strings.HasPrefix(cleaned, "data"+string(filepath.Separator)) || cleaned == "data" {
				dbPath = filepath.Join(workdir, cleaned)
			} else {
				dbPath = filepath.Join(workdir, "data", cleaned)
			}
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
			zap.S().Warnf("failed to create directory for SQLite DB: %v", err)
		}
	}

	zap.S().Infof("SQLite database path: %s", dbPath)

	pool, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: logger.New(
			zap.NewStdLog(zap.L()),
			logger.Config{
				SlowThreshold:             time.Millisecond * 200,
				LogLevel:                  common.If(config.Debug, logger.Info, logger.Silent).(logger.LogLevel), //nolint:errcheck // type assertion is safe
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})
	common.Must(err)
	common.Must(tenancy.RegisterCallbacks(pool))

	sqlDB, err := pool.DB()
	common.Must(err)

	// SQLite connection pool settings
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	return pool
}

// postgresDSN builds a safely escaped PostgreSQL URL and rejects ambiguous or
// unsafe connection settings before any network connection is attempted.
func postgresDSN(cfg config.DBConfig) (string, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" || strings.TrimSpace(cfg.Name) == "" || strings.TrimSpace(cfg.User) == "" {
		return "", fmt.Errorf("PostgreSQL host, database name, and user are required")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return "", fmt.Errorf("PostgreSQL port must be between 1 and 65535")
	}
	sslMode := strings.ToLower(strings.TrimSpace(cfg.SSLMode))
	if sslMode == "" {
		sslMode = "require"
	}
	switch sslMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return "", fmt.Errorf("unsupported PostgreSQL sslmode %q", sslMode)
	}
	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 5
	}
	if timeout < 1 || timeout > 60 {
		return "", fmt.Errorf("PostgreSQL connect timeout must be between 1 and 60 seconds")
	}
	if (strings.TrimSpace(cfg.SSLCert) == "") != (strings.TrimSpace(cfg.SSLKey) == "") {
		return "", fmt.Errorf("PostgreSQL client certificate and key must be configured together")
	}

	query := url.Values{}
	query.Set("dbname", cfg.Name)
	query.Set("sslmode", sslMode)
	query.Set("connect_timeout", strconv.Itoa(timeout))
	query.Set("TimeZone", "Asia/Jakarta")
	if root := strings.TrimSpace(cfg.SSLRootCert); root != "" {
		query.Set("sslrootcert", root)
	}
	if cert := strings.TrimSpace(cfg.SSLCert); cert != "" {
		query.Set("sslcert", cert)
		query.Set("sslkey", strings.TrimSpace(cfg.SSLKey))
	}
	u := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(cfg.User, cfg.Passwd),
		Host: net.JoinHostPort(host, strconv.Itoa(cfg.Port)),
		RawQuery: query.Encode(),
	}
	return u.String(), nil
}

// getPgDatabase returns a verified PostgreSQL connection or a startup error.
// Connection and initial ping are bounded by ConnectTimeout.
func getPgDatabase(cfg config.DBConfig) (*gorm.DB, error) {
	dsn, err := postgresDSN(cfg)
	if err != nil {
		return nil, err
	}
	pool, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction: true,
		PrepareStmt: true,
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger: logger.New(
			zap.NewStdLog(zap.L()),
			logger.Config{
				SlowThreshold: time.Millisecond * 200,
				LogLevel: common.If(cfg.Debug, logger.Info, logger.Silent).(logger.LogLevel), //nolint:errcheck
				IgnoreRecordNotFoundError: true,
				Colorful: false,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	if err := tenancy.RegisterCallbacks(pool); err != nil {
		return nil, fmt.Errorf("register PostgreSQL tenant scopes: %w", err)
	}
	sqlDB, err := pool.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 5
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("PostgreSQL health check failed: %w", err)
	}
	sqlDB.SetMaxIdleConns(cfg.IdleConn)
	sqlDB.SetMaxOpenConns(cfg.MaxConn)
	return pool, nil
}