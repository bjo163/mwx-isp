package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLooksScratch(t *testing.T) {
	tests := []struct {
		dsn  string
		want bool
	}{
		{"host=127.0.0.1 dbname=mwxbench user=mwx", true},
		{"host=127.0.0.1 dbname=mwx_test", true},
		{"host=127.0.0.1 dbname=scratch_accounting", true},
		{"host=127.0.0.1 dbname=mwx_isp user=mwx", false},
		{"postgres://user:pass@db:5432/production", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, looksScratch(tt.dsn), tt.dsn)
	}
}

func TestBuildQueries(t *testing.T) {
	queries := buildQueries()
	require.Len(t, queries, 7)

	names := make(map[string]bool, len(queries))
	for _, q := range queries {
		require.NotEmpty(t, q.sql, q.name)
		require.False(t, names[q.name], "duplicate query name %q", q.name)
		names[q.name] = true
		require.NotEmpty(t, q.args, "query %q has no bound parameters", q.name)
	}

	assert.True(t, names["retention_delete_plan"])
	for _, q := range queries {
		if q.name == "retention_delete_plan" {
			assert.True(t, q.explainOnly, "the DELETE must be planned, never executed")
			continue
		}
		assert.False(t, q.explainOnly, "query %q must be executed", q.name)
	}
}
