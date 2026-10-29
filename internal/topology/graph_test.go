package topology

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func nasFixture(id, tenant int64, name, hostname, identifier, ip string) domain.NetNas {
	return domain.NetNas{ID: id, TenantID: tenant, Name: name, Hostname: hostname, Identifier: identifier, Ipaddr: ip}
}

func odpFixture(id, tenant int64, code, oltName, zone string, ports int) domain.ODP {
	return domain.ODP{ID: id, TenantID: tenant, Code: code, OLTName: oltName, Zone: zone, TotalPorts: ports, Status: "active"}
}

func customerFixture(id, tenant, odpID int64, port int, status string) domain.Customer {
	return domain.Customer{
		ID: id, TenantID: tenant, CustomerNo: fmt.Sprintf("CUST-%d", id), Name: fmt.Sprintf("customer %d", id),
		ODPID: odpID, ODPPort: port, Status: status,
	}
}

func findingKinds(graph Graph) []string {
	kinds := make([]string, 0, len(graph.Findings))
	for _, finding := range graph.Findings {
		kinds = append(kinds, finding.Kind)
	}
	return kinds
}

func TestBuildLinksODPByOLTName(t *testing.T) {
	graph := Build(
		[]domain.NetNas{nasFixture(1, 1, "OLT-HUAWEI-01", "olt-01.lab", "nas-01", "192.0.2.1")},
		[]domain.ODP{odpFixture(10, 1, "ODP-KNG-001", " olt-huawei-01 ", "Kuningan", 8)},
		nil,
	)

	require.Len(t, graph.Edges, 1)
	assert.Equal(t, int64(1), graph.Edges[0].From)
	assert.Equal(t, int64(10), graph.Edges[0].To)
	assert.Equal(t, "name", graph.Edges[0].MatchField)
	assert.False(t, graph.Edges[0].Ambiguous)
	assert.Equal(t, 1, graph.Stats.Linked)
	assert.Zero(t, graph.Stats.Unlinked)
	assert.Empty(t, graph.Findings, "%v", graph.Findings)
}

func TestBuildFallsBackToHostnameAndIdentifier(t *testing.T) {
	graph := Build(
		[]domain.NetNas{
			nasFixture(1, 1, "edge-router", "OLT-02.LAB", "", "192.0.2.2"),
			nasFixture(2, 1, "bras", "", "bras-03", "192.0.2.3"),
		},
		[]domain.ODP{
			odpFixture(10, 1, "ODP-A", "olt-02.lab", "Zone A", 8),
			odpFixture(11, 1, "ODP-B", "BRAS-03", "Zone B", 8),
		},
		nil,
	)

	require.Len(t, graph.Edges, 2)
	assert.Equal(t, "hostname", graph.Edges[0].MatchField)
	assert.Equal(t, int64(10), graph.Edges[0].To)
	assert.Equal(t, "identifier", graph.Edges[1].MatchField)
	assert.Equal(t, int64(11), graph.Edges[1].To)
	assert.Empty(t, graph.Findings, "%v", graph.Findings)
}

func TestBuildReportsUnlinkedAndEmptyOLTName(t *testing.T) {
	graph := Build(
		[]domain.NetNas{nasFixture(1, 1, "OLT-HUAWEI-01", "", "", "192.0.2.1")},
		[]domain.ODP{
			odpFixture(10, 1, "ODP-EMPTY", "", "Zone A", 8),
			odpFixture(11, 1, "ODP-TYPO", "OLT-HUAWEI-1", "Zone B", 8),
		},
		nil,
	)

	assert.Empty(t, graph.Edges)
	assert.Equal(t, 2, graph.Stats.Unlinked)
	assert.Equal(t, []string{FindingODPUnlinked, FindingODPUnlinked}, findingKinds(graph), "%v", graph.Findings)
}

func TestBuildReportsAmbiguousOLTName(t *testing.T) {
	graph := Build(
		[]domain.NetNas{
			nasFixture(1, 1, "OLT-SAME", "", "", "192.0.2.1"),
			nasFixture(2, 1, "olt-same", "", "", "192.0.2.2"),
		},
		[]domain.ODP{odpFixture(10, 1, "ODP-A", "OLT-SAME", "Zone A", 8)},
		nil,
	)

	require.Len(t, graph.Edges, 2)
	assert.True(t, graph.Edges[0].Ambiguous)
	assert.True(t, graph.Edges[1].Ambiguous)
	assert.Equal(t, 1, graph.Stats.Ambiguous)
	assert.Equal(t, []string{FindingOLTNameAmbiguous}, findingKinds(graph), "%v", graph.Findings)
}

func TestBuildDoesNotCrossTenant(t *testing.T) {
	graph := Build(
		[]domain.NetNas{nasFixture(2, 2, "OLT-SHARED", "", "", "198.51.100.1")},
		[]domain.ODP{odpFixture(10, 1, "ODP-KNG-001", "OLT-SHARED", "Kuningan", 8)},
		nil,
	)

	assert.Empty(t, graph.Edges)
	assert.Equal(t, []string{FindingODPUnlinked, FindingOLTNameCrossTenant}, findingKinds(graph), "%v", graph.Findings)
}

func TestBuildCountsDistinctPortsAndOverflow(t *testing.T) {
	graph := Build(
		[]domain.NetNas{nasFixture(1, 1, "OLT-1", "", "", "192.0.2.1")},
		[]domain.ODP{
			odpFixture(10, 1, "ODP-A", "OLT-1", "Zone A", 8),
			odpFixture(11, 1, "ODP-B", "OLT-1", "Zone B", 1),
		},
		[]domain.Customer{
			customerFixture(1, 1, 10, 1, domain.CustomerActive),
			customerFixture(2, 1, 10, 1, domain.CustomerSuspended),
			customerFixture(3, 1, 10, 2, domain.CustomerActive),
			customerFixture(4, 1, 10, 3, domain.CustomerTerminated),
			customerFixture(5, 1, 11, 1, domain.CustomerActive),
			customerFixture(6, 1, 11, 2, domain.CustomerActive),
			customerFixture(7, 1, 0, 0, domain.CustomerActive),
		},
	)

	assert.Equal(t, 2, graph.ODPs[0].PortsUsed, "terminated customers release their port and duplicate ports count once")
	assert.Equal(t, 2, graph.ODPs[1].PortsUsed)
	assert.Equal(t, 1, graph.Stats.Unassigned)
	assert.Equal(t, []string{FindingODPPortOverflow}, findingKinds(graph), "%v", graph.Findings)
}

func TestRenderSVGIsDeterministicAndEscaped(t *testing.T) {
	graph := Build(
		[]domain.NetNas{
			nasFixture(1, 1, `<script>alert("x")</script>`, "", "", "192.0.2.1"),
			nasFixture(2, 1, "OLT-DUP", "", "", "192.0.2.2"),
			nasFixture(3, 1, "OLT-DUP", "", "", "192.0.2.3"),
		},
		[]domain.ODP{
			odpFixture(10, 1, "ODP-A", "OLT-DUP", "Zone A", 8),
			odpFixture(11, 1, "ODP-B", "OLT-UNKNOWN", "Zone B", 8),
		},
		nil,
	)

	first := RenderSVG(graph)
	second := RenderSVG(graph)

	assert.Equal(t, first, second)
	assert.True(t, strings.HasPrefix(first, "<svg"))
	assert.True(t, strings.HasSuffix(first, "</svg>\n"))
	assert.NotContains(t, first, "<script>")
	assert.Contains(t, first, "&lt;script&gt;")
	assert.Contains(t, first, `stroke-dasharray="6 4"`, "ambiguous edges and unlinked boxes are dashed")
	assert.Contains(t, first, "ODP-B (unlinked)")
}

var topologyDBSeq atomic.Int64

func TestBuildFromDBReadsStoredRows(t *testing.T) {
	dsn := fmt.Sprintf("file:topology-%d?mode=memory&cache=shared", topologyDBSeq.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.NetNas{}, &domain.ODP{}, &domain.Customer{}))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	require.NoError(t, db.Create(&domain.NetNas{ID: 1, TenantID: 1, Name: "OLT-HUAWEI-01", Ipaddr: "192.0.2.1"}).Error)
	require.NoError(t, db.Create(&domain.ODP{ID: 10, TenantID: 1, Code: "ODP-KNG-001", OLTName: "olt-huawei-01", TotalPorts: 8}).Error)
	require.NoError(t, db.Create(&domain.Customer{ID: 1, TenantID: 1, CustomerNo: "CUST-1", Name: "customer", ODPID: 10, ODPPort: 1, Status: domain.CustomerActive}).Error)

	graph, err := BuildFromDB(db)
	require.NoError(t, err)

	require.Len(t, graph.Edges, 1)
	assert.Equal(t, int64(10), graph.Edges[0].To)
	assert.Equal(t, 1, graph.ODPs[0].PortsUsed)
}
