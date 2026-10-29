// Package topology builds the static NAS-to-ODP graph implied by the relations
// an operator already filled in and draws it as a self-contained SVG.
//
// This is the bounded FUT-006 prototype: no application package imports it and
// it never writes. It exists to answer whether the stored relations are strong
// enough to render a topology map without inventing edges.
//
// The only edge an operator can enter is an ODP's OLT name. MWX-ISP stores no
// ODP-to-NAS foreign key, so the graph matches that free-text name against the
// NAS name, hostname and identifier inside one tenant. A name that matches
// nothing, matches several NAS, or matches only another tenant's NAS is
// reported instead of drawn as a link.
package topology

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bjo163/mwx-isp/internal/domain"
	"gorm.io/gorm"
)

// Finding kinds reported by Build.
const (
	// FindingODPUnlinked reports an ODP whose OLT name matches no NAS in its
	// tenant, including an empty name.
	FindingODPUnlinked = "odp_unlinked"
	// FindingOLTNameAmbiguous reports an OLT name that matches more than one
	// NAS in the tenant on the same field.
	FindingOLTNameAmbiguous = "olt_name_ambiguous"
	// FindingOLTNameCrossTenant reports an OLT name that matches only a NAS in a
	// different tenant. The graph does not cross the tenant boundary.
	FindingOLTNameCrossTenant = "olt_name_cross_tenant"
	// FindingODPPortOverflow reports more occupied ports than the ODP declares.
	FindingODPPortOverflow = "odp_port_overflow"
)

// Match fields in priority order. An ODP whose OLT name matches a NAS name is
// reported as a name match even when the same value also matches that NAS's
// hostname.
var matchFields = []string{"name", "hostname", "identifier"}

// Node is one device in the graph.
type Node struct {
	// Kind is "nas" or "odp".
	Kind string
	// ID is the row id, unique within its kind.
	ID int64
	// TenantID is carried so a caller can group nodes without re-querying.
	TenantID int64
	// Label is the operator-facing name: the NAS name or the ODP code.
	Label string
	// Detail is the NAS address or the ODP zone.
	Detail string
	// Ports and PortsUsed are set for ODP nodes only. PortsUsed counts distinct
	// numbered ports held by non-terminated customers.
	Ports     int
	PortsUsed int
}

// Edge links one NAS to one ODP through a matched field value.
type Edge struct {
	// From is the NAS id and To the ODP id.
	From int64
	To   int64
	// MatchField is the NAS field the OLT name matched: name, hostname or
	// identifier.
	MatchField string
	// MatchValue is the OLT name as the operator entered it.
	MatchValue string
	// Ambiguous is true when more than one NAS in the tenant matched, so the
	// edge is a candidate rather than a fact.
	Ambiguous bool
}

// Finding is one relation the graph cannot draw honestly.
type Finding struct {
	// Kind is one of the Finding* constants.
	Kind string
	// Subject is the ODP code the finding is about.
	Subject string
	// Detail states the entered name and what it matched.
	Detail string
}

// Stats counts what the graph contains.
type Stats struct {
	// NAS and ODPs count the devices.
	NAS  int
	ODPs int
	// Linked and Unlinked count ODPs by whether they have at least one edge.
	Linked   int
	Unlinked int
	// Ambiguous counts ODPs with more than one candidate NAS.
	Ambiguous int
	// Customers and Unassigned count the customer rows that were read, and the
	// ones with no ODP.
	Customers  int
	Unassigned int
}

// Graph is a static NAS-to-ODP map.
type Graph struct {
	// NAS and ODPs are ordered by tenant then id.
	NAS  []Node
	ODPs []Node
	// Edges are ordered by NAS id then ODP id.
	Edges []Edge
	// Findings are ordered by ODP code then kind.
	Findings []Finding
	Stats    Stats
}

// matchKey indexes one NAS field value inside one tenant.
type matchKey struct {
	tenant int64
	field  string
	value  string
}

// Build links every ODP to the NAS named by its OLT name and reports the links
// that cannot be drawn.
//
// Matching is case-insensitive after trimming, uses only rows of the same
// tenant, and tries the NAS name first, then hostname, then identifier. When the
// winning field matches several NAS the edges are kept as candidates with
// Ambiguous set, because choosing one would fabricate a topology that no
// operator entered.
func Build(nas []domain.NetNas, odps []domain.ODP, customers []domain.Customer) Graph {
	graph := Graph{
		NAS:  make([]Node, 0, len(nas)),
		ODPs: make([]Node, 0, len(odps)),
		Stats: Stats{
			NAS:       len(nas),
			ODPs:      len(odps),
			Customers: len(customers),
		},
	}

	index := make(map[matchKey][]domain.NetNas, len(nas)*len(matchFields))
	for _, device := range nas {
		for _, field := range matchFields {
			value := normalize(nasField(device, field))
			if value == "" {
				continue
			}
			k := matchKey{tenant: device.TenantID, field: field, value: value}
			index[k] = append(index[k], device)
		}
		graph.NAS = append(graph.NAS, Node{
			Kind:     "nas",
			ID:       device.ID,
			TenantID: device.TenantID,
			Label:    device.Name,
			Detail:   device.Ipaddr,
		})
	}
	sort.SliceStable(graph.NAS, func(i, j int) bool {
		if graph.NAS[i].TenantID != graph.NAS[j].TenantID {
			return graph.NAS[i].TenantID < graph.NAS[j].TenantID
		}
		return graph.NAS[i].ID < graph.NAS[j].ID
	})

	usedPorts := make(map[int64]map[int]struct{}, len(odps))
	for _, customer := range customers {
		if customer.ODPID == 0 || customer.ODPPort <= 0 || customer.Status == domain.CustomerTerminated {
			continue
		}
		if usedPorts[customer.ODPID] == nil {
			usedPorts[customer.ODPID] = make(map[int]struct{})
		}
		usedPorts[customer.ODPID][customer.ODPPort] = struct{}{}
	}
	unassigned := 0
	for _, customer := range customers {
		if customer.ODPID == 0 {
			unassigned++
		}
	}
	graph.Stats.Unassigned = unassigned

	linked := make(map[int64]bool, len(odps))
	ambiguous := make(map[int64]bool)
	for _, odp := range odps {
		ports := len(usedPorts[odp.ID])
		graph.ODPs = append(graph.ODPs, Node{
			Kind:      "odp",
			ID:        odp.ID,
			TenantID:  odp.TenantID,
			Label:     odp.Code,
			Detail:    odp.Zone,
			Ports:     odp.TotalPorts,
			PortsUsed: ports,
		})
		if odp.TotalPorts > 0 && ports > odp.TotalPorts {
			graph.Findings = append(graph.Findings, Finding{
				Kind:    FindingODPPortOverflow,
				Subject: odp.Code,
				Detail:  fmt.Sprintf("%d ports are occupied but the ODP declares %d", ports, odp.TotalPorts),
			})
		}

		name := strings.TrimSpace(odp.OLTName)
		if name == "" {
			graph.Findings = append(graph.Findings, Finding{
				Kind:    FindingODPUnlinked,
				Subject: odp.Code,
				Detail:  "the ODP has no OLT name, so no NAS can be derived",
			})
			continue
		}
		normalized := normalize(name)
		var matched []domain.NetNas
		field := ""
		for _, candidate := range matchFields {
			matched = index[matchKey{tenant: odp.TenantID, field: candidate, value: normalized}]
			if len(matched) > 0 {
				field = candidate
				break
			}
		}
		if len(matched) == 0 {
			other := 0
			for _, candidate := range matchFields {
				other += len(indexAcrossTenants(index, candidate, normalized))
			}
			detail := fmt.Sprintf("OLT name %q matches no NAS in tenant %d", name, odp.TenantID)
			if other > 0 {
				detail = fmt.Sprintf("OLT name %q matches only a NAS in another tenant", name)
				graph.Findings = append(graph.Findings, Finding{
					Kind:    FindingOLTNameCrossTenant,
					Subject: odp.Code,
					Detail:  detail,
				})
			}
			graph.Findings = append(graph.Findings, Finding{
				Kind:    FindingODPUnlinked,
				Subject: odp.Code,
				Detail:  detail,
			})
			continue
		}

		for _, device := range matched {
			link := Edge{
				From:       device.ID,
				To:         odp.ID,
				MatchField: field,
				MatchValue: name,
				Ambiguous:  len(matched) > 1,
			}
			graph.Edges = append(graph.Edges, link)
		}
		linked[odp.ID] = true
		if len(matched) > 1 {
			ambiguous[odp.ID] = true
			labels := make([]string, 0, len(matched))
			for _, device := range matched {
				labels = append(labels, fmt.Sprintf("%s (id %d)", device.Name, device.ID))
			}
			graph.Findings = append(graph.Findings, Finding{
				Kind:    FindingOLTNameAmbiguous,
				Subject: odp.Code,
				Detail:  fmt.Sprintf("OLT name %q matches %d NAS by %s: %s", name, len(matched), field, strings.Join(labels, ", ")),
			})
		}
	}
	sort.SliceStable(graph.ODPs, func(i, j int) bool {
		if graph.ODPs[i].TenantID != graph.ODPs[j].TenantID {
			return graph.ODPs[i].TenantID < graph.ODPs[j].TenantID
		}
		return graph.ODPs[i].ID < graph.ODPs[j].ID
	})
	sort.SliceStable(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From != graph.Edges[j].From {
			return graph.Edges[i].From < graph.Edges[j].From
		}
		return graph.Edges[i].To < graph.Edges[j].To
	})
	sort.SliceStable(graph.Findings, func(i, j int) bool {
		if graph.Findings[i].Subject != graph.Findings[j].Subject {
			return graph.Findings[i].Subject < graph.Findings[j].Subject
		}
		return graph.Findings[i].Kind < graph.Findings[j].Kind
	})

	graph.Stats.Linked = len(linked)
	graph.Stats.Unlinked = len(odps) - len(linked)
	graph.Stats.Ambiguous = len(ambiguous)
	return graph
}

// indexAcrossTenants returns the entries of one field across every tenant, so a
// near-miss that only crosses the tenant boundary can be named.
func indexAcrossTenants(index map[matchKey][]domain.NetNas, field, value string) []domain.NetNas {
	var devices []domain.NetNas
	for k, entries := range index {
		if k.field == field && k.value == value {
			devices = append(devices, entries...)
		}
	}
	return devices
}

// BuildFromDB loads NAS, ODP and customer rows and builds the graph. It is
// read-only.
func BuildFromDB(db *gorm.DB) (Graph, error) {
	var nas []domain.NetNas
	if err := db.Order("tenant_id, id").Find(&nas).Error; err != nil {
		return Graph{}, err
	}
	var odps []domain.ODP
	if err := db.Order("tenant_id, id").Find(&odps).Error; err != nil {
		return Graph{}, err
	}
	var customers []domain.Customer
	if err := db.Order("tenant_id, id").Find(&customers).Error; err != nil {
		return Graph{}, err
	}
	return Build(nas, odps, customers), nil
}

func nasField(device domain.NetNas, field string) string {
	switch field {
	case "hostname":
		return device.Hostname
	case "identifier":
		return device.Identifier
	default:
		return device.Name
	}
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
