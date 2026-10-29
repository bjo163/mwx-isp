package policycheck

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeScopeFixtures(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		unscoped  []string
	}{
		{
			name:      "single table with a bound tenant predicate",
			statement: `SELECT * FROM radius_online WHERE tenant_id = ?`,
			unscoped:  nil,
		},
		{
			name:      "single table without any predicate",
			statement: `SELECT * FROM radius_online`,
			unscoped:  []string{"radius_online"},
		},
		{
			name:      "tenant_id only in a comment is not a predicate",
			statement: "SELECT * FROM radius_online -- TODO tenant_id = 0\n",
			unscoped:  []string{"radius_online"},
		},
		{
			name:      "tenant_id in ORDER BY is not a predicate",
			statement: `SELECT username FROM radius_online ORDER BY tenant_id`,
			unscoped:  []string{"radius_online"},
		},
		{
			name: "join predicate on the driving table scopes the joined tables",
			statement: `SELECT * FROM isp_subscription AS s
				JOIN isp_customer AS c ON c.id = s.customer_id AND c.tenant_id = s.tenant_id
				LEFT JOIN radius_user AS u ON u.id = s.radius_user_id AND u.tenant_id = s.tenant_id
				WHERE s.tenant_id = ?`,
			unscoped: nil,
		},
		{
			name: "a join between two unscoped tables stays unscoped",
			statement: `SELECT * FROM radius_online AS o
				LEFT JOIN radius_user AS u ON u.username = o.username AND u.tenant_id = o.tenant_id`,
			unscoped: []string{"radius_online AS o", "radius_user AS u"},
		},
		{
			name: "equality to a scoped table reaches a third table",
			statement: `SELECT * FROM isp_trouble_ticket AS t
				JOIN isp_customer AS c ON c.id = t.customer_id AND c.tenant_id = t.tenant_id
				JOIN isp_invoice AS i ON i.customer_id = c.id AND i.tenant_id = c.tenant_id
				WHERE t.tenant_id = ?`,
			unscoped: nil,
		},
		{
			name: "a relational edge from an unscoped table does not scope its neighbor",
			statement: `SELECT * FROM radius_online AS o
				JOIN radius_accounting AS a ON a.username = o.username AND a.tenant_id = o.tenant_id
				WHERE o.username = ?`,
			unscoped: []string{"radius_online AS o", "radius_accounting AS a"},
		},
		{
			name:      "bare tenant_id is ambiguous once a second table appears",
			statement: `SELECT * FROM isp_invoice JOIN isp_customer ON isp_customer.id = isp_invoice.customer_id WHERE tenant_id = ?`,
			unscoped:  []string{"isp_invoice", "isp_customer"},
		},
		{
			name:      "delete with a bound tenant predicate",
			statement: `DELETE FROM radius_online WHERE tenant_id = ?`,
			unscoped:  nil,
		},
		{
			name:      "delete without any predicate",
			statement: `DELETE FROM radius_online WHERE username = ?`,
			unscoped:  []string{"radius_online"},
		},
		{
			name:      "update without any predicate",
			statement: `UPDATE isp_invoice SET status = ? WHERE id = ?`,
			unscoped:  []string{"isp_invoice"},
		},
		{
			name:      "quoted identifiers and schema prefixes are normalised",
			statement: "SELECT * FROM \"public\".\"Radius_Online\" WHERE \"radius_online\".\"tenant_id\" = $1",
			unscoped:  nil,
		},
		{
			name:      "a subquery's tables are left to the owning statement",
			statement: `SELECT * FROM radius_online AS o WHERE o.username IN (SELECT username FROM radius_accounting WHERE tenant_id = ?)`,
			unscoped:  nil,
		},
		{
			name:      "a statement without tables is not flagged",
			statement: `SELECT setval('tenant_id_seq', 1)`,
			unscoped:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Analyze(tt.statement)
			assert.Equal(t, tt.unscoped, report.Unscoped)
			assert.Equal(t, tt.unscoped == nil, report.Scoped())
		})
	}
}

func TestScanFileSeparatesLiteralFromDynamic(t *testing.T) {
	src := []byte(`package fixture
func f(db *gorm.DB, sql string, ctx context.Context) {
	db.Table("isp_customer").Scan(nil)
	// tenancy:unscoped: platform report, cross-tenant by design
	db.Table("isp_customer").Scan(nil)
	db.Raw(sql).Scan(nil)
	db.Table("radius_online").Where("tenant_id = ?", 1).Scan(nil)
}
`)
	sites, err := scanFile("fixture.go", src)
	require.NoError(t, err)
	require.Len(t, sites, 4)

	assert.False(t, sites[0].Dynamic)
	assert.False(t, sites[0].Annotated)
	assert.Equal(t, []string{"isp_customer"}, sites[0].Report.Unscoped)

	assert.True(t, sites[1].Annotated)

	assert.True(t, sites[2].Dynamic)
	assert.Empty(t, sites[2].Report.Unscoped)

	assert.True(t, sites[3].Report.Scoped())

	assert.Len(t, Findings(sites), 1, "only the unannotated, unscoped literal statement is a finding")
}

// TestRepoRawSQLStatementsSatisfyTheTablePolicy is the survey behind FUT-010:
// every literal raw-SQL statement in production code either bounds each
// referenced relation or explains the exemption. `go test -v` prints the
// classification for the memo.
func TestRepoRawSQLStatementsSatisfyTheTablePolicy(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	sites, err := Scan(root)
	require.NoError(t, err)
	require.NotEmpty(t, sites)

	literal, dynamic, annotated := 0, 0, 0
	for _, site := range sites {
		switch {
		case site.Dynamic:
			dynamic++
			t.Logf("%s:%d unanalysable: %s", site.File, site.Line, site.SkipReason)
		default:
			literal++
			if site.Annotated {
				annotated++
			}
			t.Logf("%s:%d tables=%v unscoped=%v annotated=%t",
				site.File, site.Line, site.Report.Tables, site.Report.Unscoped, site.Annotated)
		}
	}
	t.Logf("literal=%d dynamic=%d annotated=%d", literal, dynamic, annotated)

	for _, finding := range Findings(sites) {
		t.Errorf("%s:%d: unscoped relations %v; add a tenant predicate or //%s <reason>",
			finding.File, finding.Line, finding.Report.Unscoped, Annotation)
	}
}
