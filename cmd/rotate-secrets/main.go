/*
 * MWX-ISP SNMP secret rotation tool
 *
 * Re-encrypts the stored SNMPv2c communities and SNMPv3 auth/privacy secrets
 * from an old web secret to a new one. Run with no flags for a dry run (the
 * updates are rolled back) and add -apply to persist them.
 *
 * Usage:
 *   MWX_ISP_OLD_WEB_SECRET=... MWX_ISP_NEW_WEB_SECRET=... go run ./cmd/rotate-secrets -c mwx-isp.yml
 *   MWX_ISP_OLD_WEB_SECRET=... MWX_ISP_NEW_WEB_SECRET=... go run ./cmd/rotate-secrets -c mwx-isp.yml -apply
 *
 * The secrets are read from the environment so they never appear in the
 * process list or shell history, and the output contains counts only.
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/bjo163/mwx-isp/config"
	"github.com/bjo163/mwx-isp/internal/networkmonitor"
)

const (
	oldSecretEnvVar = "MWX_ISP_OLD_WEB_SECRET"
	newSecretEnvVar = "MWX_ISP_NEW_WEB_SECRET"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("rotate-secrets", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configFile := flags.String("c", "mwx-isp.yml", "configuration file path")
	apply := flags.Bool("apply", false, "persist the re-encrypted credentials (default is a dry run)")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	oldSecret, newSecret := getenv(oldSecretEnvVar), getenv(newSecretEnvVar)
	if strings.TrimSpace(oldSecret) == "" || strings.TrimSpace(newSecret) == "" {
		fmt.Fprintf(stderr, "Error: set %s and %s to the old and new web secret\n", oldSecretEnvVar, newSecretEnvVar)
		return 2
	}

	if *apply {
		fmt.Fprintln(stdout, "Stop the service before applying and restart it with the new web secret afterwards.")
	}

	cfg := config.LoadConfig(*configFile)
	db, err := openDatabase(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "Error: connect to database: %v\n", err)
		return 1
	}
	if sqlDB, err := db.DB(); err == nil {
		defer func() { _ = sqlDB.Close() }()
	}

	report, err := networkmonitor.RotateSNMPSecrets(context.Background(), db, oldSecret, newSecret, *apply)
	if err != nil {
		fmt.Fprintf(stderr, "Error: rotation aborted, no row was changed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "ok apply=%t targets=%d fields=%d\n", *apply, report.Targets, report.Fields)
	if !*apply {
		fmt.Fprintln(stdout, "dry run: writes were rolled back; re-run with -apply to persist them.")
	}
	return 0
}

func openDatabase(cfg *config.AppConfig) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: logger.New(nil, logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
		}),
	}

	switch dbType := strings.ToLower(cfg.Database.Type); dbType {
	case "sqlite":
		dbPath := cfg.Database.Name
		if dbPath != ":memory:" && !path.IsAbs(dbPath) {
			dbPath = path.Join(cfg.System.Workdir, "data", dbPath)
		}
		return gorm.Open(sqlite.Open(dbPath), gormCfg)
	case "postgres", "postgresql":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
			cfg.Database.Host, cfg.Database.User, cfg.Database.Passwd, cfg.Database.Name, cfg.Database.Port)
		return gorm.Open(postgres.Open(dsn), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
