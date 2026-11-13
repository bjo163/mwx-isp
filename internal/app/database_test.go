package app

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/bjo163/mwx-isp/config"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDatabaseRespectsPlatformAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absolute-test.db")
	db, err := getDatabase(config.DBConfig{Type: "sqlite", Name: path}, t.TempDir())
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.Exec("CREATE TABLE probe (id INTEGER PRIMARY KEY)").Error)
	_, err = os.Stat(path)
	require.NoError(t, err)
}


func TestPostgresFailureDoesNotCreateSQLiteFallback(t *testing.T) {
	workdir := t.TempDir()
	db, err := getDatabase(config.DBConfig{
		Type: "postgres", Host: "127.0.0.1", Port: 1,
		Name: "mwxisp", User: "mwxisp", SSLMode: "disable", ConnectTimeout: 1,
	}, workdir)
	require.Error(t, err)
	require.Nil(t, db)
	_, statErr := os.Stat(filepath.Join(workdir, "data", "mwx-isp.db"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestPostgresDSNUsesConfiguredTLSAndEscapesCredentials(t *testing.T) {
	dsn, err := postgresDSN(config.DBConfig{
		Host: "db.example.test", Port: 5432, Name: "isp database",
		User: "isp user", Passwd: "p@ss:word", SSLMode: "verify-full",
		SSLRootCert: "/run/secrets/pg-ca.crt", ConnectTimeout: 7,
	})
	require.NoError(t, err)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "isp user", parsed.User.Username())
	password, _ := parsed.User.Password()
	require.Equal(t, "p@ss:word", password)
	require.Equal(t, "isp database", parsed.Query().Get("dbname"))
	require.Equal(t, "verify-full", parsed.Query().Get("sslmode"))
	require.Equal(t, "/run/secrets/pg-ca.crt", parsed.Query().Get("sslrootcert"))
	require.Equal(t, "7", parsed.Query().Get("connect_timeout"))
}

func TestPostgresDSNRejectsInvalidTLSConfiguration(t *testing.T) {
	_, err := postgresDSN(config.DBConfig{
		Host: "db", Port: 5432, Name: "isp", User: "isp",
		SSLMode: "unsafe", ConnectTimeout: 5,
	})
	require.Error(t, err)
	_, err = postgresDSN(config.DBConfig{
		Host: "db", Port: 5432, Name: "isp", User: "isp",
		SSLMode: "require", SSLCert: "/client.crt", ConnectTimeout: 5,
	})
	require.Error(t, err)
}
