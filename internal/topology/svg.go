package topology

import (
	"fmt"
	"html"
	"strings"
)

// SVG layout constants. The drawing is a fixed two-column map: NAS devices on
// the left, ODPs on the right, one row per device.
const (
	svgWidth  = 960
	svgLeft   = 40
	svgRight  = 640
	svgBoxW   = 280
	svgBoxH   = 36
	svgRowH   = 56
	svgTop    = 116
	svgBottom = 96
)

// RenderSVG draws the graph as one self-contained static SVG document with no
// script, no external reference and no embedded data beyond the labels.
//
// The output is deterministic for the same graph: edges are drawn in graph
// order, each unlinked ODP gets an amber dashed border, and each ambiguous edge
// is dashed so a candidate link is never mistaken for a fact. Labels are XML
// escaped and clipped so an operator-entered name cannot break the document.
func RenderSVG(graph Graph) string {
	rows := len(graph.NAS)
	if len(graph.ODPs) > rows {
		rows = len(graph.ODPs)
	}
	height := svgTop + rows*svgRowH + svgBottom

	nasRow := make(map[int64]int, len(graph.NAS))
	for i, node := range graph.NAS {
		nasRow[node.ID] = i
	}
	odpRow := make(map[int64]int, len(graph.ODPs))
	for i, node := range graph.ODPs {
		odpRow[node.ID] = i
	}
	linked := make(map[int64]bool, len(graph.ODPs))
	for _, edge := range graph.Edges {
		linked[edge.To] = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="ui-monospace, monospace" font-size="12">`+"\n",
		svgWidth, height, svgWidth, height)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#0f1720"/>`+"\n", svgWidth, height)
	fmt.Fprintf(&b, `<text x="%d" y="36" fill="#e2e8f0" font-size="18">NAS &#8594; ODP static graph</text>`+"\n", svgLeft)
	fmt.Fprintf(&b, `<text x="%d" y="60" fill="#94a3b8">%d NAS, %d ODP, %d linked, %d unlinked, %d ambiguous, %d customers (%d without ODP)</text>`+"\n",
		svgLeft, graph.Stats.NAS, graph.Stats.ODPs, graph.Stats.Linked, graph.Stats.Unlinked,
		graph.Stats.Ambiguous, graph.Stats.Customers, graph.Stats.Unassigned)
	fmt.Fprintf(&b, `<text x="%d" y="84" fill="#64748b">solid: matched OLT name &#183; dashed edge: ambiguous &#183; dashed box: unlinked</text>`+"\n", svgLeft)
	fmt.Fprintf(&b, `<text x="%d" y="84" fill="#64748b">NAS</text>`+"\n", svgLeft)
	fmt.Fprintf(&b, `<text x="%d" y="84" fill="#64748b">ODP</text>`+"\n", svgRight)

	for _, edge := range graph.Edges {
		fromRow, fromOK := nasRow[edge.From]
		toRow, toOK := odpRow[edge.To]
		if !fromOK || !toOK {
			continue
		}
		style := ""
		if edge.Ambiguous {
			style = ` stroke-dasharray="6 4"`
		}
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#3b82f6" stroke-width="1.5"%s/>`+"\n",
			svgLeft+svgBoxW, boxCenter(fromRow), svgRight, boxCenter(toRow), style)
	}

	for i, node := range graph.NAS {
		y := boxTop(i)
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="#1e293b" stroke="#475569"/>`+"\n",
			svgLeft, y, svgBoxW, svgBoxH)
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#e2e8f0">%s</text>`+"\n", svgLeft+12, y+16, escape(clip(node.Label, 30)))
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#94a3b8" font-size="10">tenant %d &#183; %s</text>`+"\n",
			svgLeft+12, y+30, node.TenantID, escape(clip(node.Detail, 36)))
	}

	for i, node := range graph.ODPs {
		y := boxTop(i)
		border := "#475569"
		dash := ""
		if !linked[node.ID] {
			border = "#f59e0b"
			dash = ` stroke-dasharray="6 4"`
		}
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="#1e293b" stroke="%s"%s/>`+"\n",
			svgRight, y, svgBoxW, svgBoxH, border, dash)
		label := node.Label
		if !linked[node.ID] {
			label += " (unlinked)"
		}
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#e2e8f0">%s</text>`+"\n", svgRight+12, y+16, escape(clip(label, 30)))
		ports := fmt.Sprintf("tenant %d", node.TenantID)
		if node.Ports > 0 {
			ports = fmt.Sprintf("%d/%d ports", node.PortsUsed, node.Ports)
		}
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#94a3b8" font-size="10">%s &#183; %s</text>`+"\n",
			svgRight+12, y+30, ports, escape(clip(node.Detail, 28)))
	}

	b.WriteString("</svg>\n")
	return b.String()
}

// boxTop and boxCenter place row i.
func boxTop(row int) int    { return svgTop + row*svgRowH }
func boxCenter(row int) int { return boxTop(row) + svgBoxH/2 }

// clip limits a label to n runes so a long operator-entered name cannot overlap
// the neighboring column.
func clip(value string, n int) string {
	runes := []rune(value)
	if len(runes) <= n {
		return value
	}
	return string(runes[:n-1]) + "\u2026"
}

func escape(value string) string {
	return html.EscapeString(value)
}
