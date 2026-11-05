package report

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/scan"
)

// SVGs use CSS classes so the embedding page controls light/dark theming.
// Standalone exports wrap them with svgStandaloneStyle.
const svgStandaloneStyle = `<style>
text{font-family:system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;fill:#1d2733}
.col{fill:#f3f5f8}.colt{font-size:12px;font-weight:600;fill:#5b6675}
.node{stroke-width:1}.nt{font-size:11px}.ns{font-size:10px;fill:#5b6675}
.e{fill:none;stroke:#8a96a6;stroke-opacity:.55}.e.bad{stroke:#c9372c;stroke-opacity:.85}
.L-Presentation{fill:#e3edfb;stroke:#4a78c2}.L-Business{fill:#e5f4ea;stroke:#3b8a55}.L-Integration{fill:#fff1dc;stroke:#c98a1e}
.L-Persistence{fill:#f6e6f3;stroke:#a24d92}.L-Model{fill:#eef0f3;stroke:#7d8794}.L-Configuration{fill:#efeafb;stroke:#6f56b8}
.L-Cross-cutting{fill:#f1f1ec;stroke:#8b8a6a}.L-Unknown{fill:#ffffff;stroke:#a7b0bc}
.sys{fill:#1f4e8c;stroke:#163a69}.syst{fill:#fff;font-size:13px;font-weight:600}
.box{fill:#fff;stroke:#a7b0bc}.bt{font-size:11px;font-weight:600}.bi{font-size:10px;fill:#5b6675}
.k-library{stroke:#8a96a6;stroke-dasharray:5 4}.k-http{stroke:#2f6fd0}.k-soap{stroke:#6f56b8}.k-jms,.k-kafka{stroke:#d98a17}
.k-ejb{stroke:#3b8a55}.k-shared-db{stroke:#c9372c;stroke-dasharray:2 3}.k-shared-datasource{stroke:#c9372c;stroke-dasharray:1 4}.k-deploys{stroke:#2a9d8f;stroke-dasharray:6 3}
.lk{fill:none;stroke-width:1.6;stroke-opacity:.8}.lk.lo{stroke-opacity:.35;stroke-width:1;stroke-dasharray:2 4}.lg{font-size:11px}
</style>`

func esc(s string) string { return html.EscapeString(s) }

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func commonPrefix(pkgs []string) string {
	if len(pkgs) < 2 {
		return ""
	}
	parts := strings.Split(pkgs[0], ".")
	n := len(parts)
	for _, p := range pkgs[1:] {
		ps := strings.Split(p, ".")
		k := 0
		for k < n && k < len(ps) && ps[k] == parts[k] {
			k++
		}
		n = k
	}
	if n == 0 {
		return ""
	}
	return strings.Join(parts[:n], ".")
}

// LayerSVG draws packages in layer columns with dependency edges.
func LayerSVG(f *model.Facts, maxNodes int, lang string) (string, string) {
	stats := append([]model.PackageStat{}, f.PackageStats...)
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Types != stats[j].Types {
			return stats[i].Types > stats[j].Types
		}
		return stats[i].Package < stats[j].Package
	})
	if len(stats) > maxNodes {
		stats = stats[:maxNodes]
	}
	if len(stats) == 0 {
		return "", ""
	}
	var names []string
	for _, s := range stats {
		names = append(names, s.Package)
	}
	prefix := commonPrefix(names)
	byLayer := map[string][]model.PackageStat{}
	for _, s := range stats {
		byLayer[s.Layer] = append(byLayer[s.Layer], s)
	}
	var cols []string
	for _, l := range scan.LayerOrder {
		if len(byLayer[l]) > 0 {
			cols = append(cols, l)
			sort.Slice(byLayer[l], func(i, j int) bool { return byLayer[l][i].Package < byLayer[l][j].Package })
		}
	}
	const colW, gap, nodeH, vgap = 200, 56, 36, 10
	type pos struct{ x, y, col int }
	colOf := map[string]int{}
	for i, l := range cols {
		colOf[l] = i
	}
	// reserve room above the nodes for arcs that skip columns
	maxSpan := 0
	compOf := map[string]model.Component{}
	for _, c := range f.Components {
		compOf[c.FQN] = c
	}
	for _, d := range f.Dependencies {
		a, b := compOf[d.From], compOf[d.To]
		if ca, ok := colOf[a.Layer]; ok {
			if cb, ok := colOf[b.Layer]; ok {
				if sp := ca - cb; sp > maxSpan || -sp > maxSpan {
					if sp < 0 {
						sp = -sp
					}
					maxSpan = sp
				}
			}
		}
	}
	top := 46
	if maxSpan > 1 {
		top += 22 * maxSpan
	}
	at := map[string]pos{}
	maxRows := 0
	for i, l := range cols {
		for j, s := range byLayer[l] {
			at[s.Package] = pos{20 + i*(colW+gap), top + j*(nodeH+vgap), i}
		}
		if len(byLayer[l]) > maxRows {
			maxRows = len(byLayer[l])
		}
	}
	w := 40 + len(cols)*(colW+gap) - gap
	h := top + maxRows*(nodeH+vgap) + 20
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" role="img" aria-label="%s">`, w, h, w, esc(tr(lang, "architecture")))
	b.WriteString(`<defs><marker id="lar" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0L10 5L0 10z" fill="#8a96a6"/></marker></defs>`)
	for i, l := range cols {
		x := 20 + i*(colW+gap)
		fmt.Fprintf(&b, `<rect class="col" x="%d" y="%d" width="%d" height="%d" rx="8"/><text class="colt" x="%d" y="%d">%s</text>`,
			x-8, top-36, colW+16, h-top+26, x, top-16, esc(tr(lang, l)))
	}
	// edges aggregated by package pair
	comp := map[string]model.Component{}
	for _, c := range f.Components {
		comp[c.FQN] = c
	}
	type edge struct {
		n   int
		bad bool
	}
	edges := map[[2]string]*edge{}
	for _, d := range f.Dependencies {
		a, b2 := comp[d.From], comp[d.To]
		if a.Package == b2.Package {
			continue
		}
		if _, ok := at[a.Package]; !ok {
			continue
		}
		if _, ok := at[b2.Package]; !ok {
			continue
		}
		k := [2]string{a.Package, b2.Package}
		if edges[k] == nil {
			edges[k] = &edge{}
		}
		edges[k].n++
		if scan.Forbidden(a.Layer, b2.Layer) {
			edges[k].bad = true
		}
	}
	keys := make([][2]string, 0, len(edges))
	for k := range edges {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+keys[i][1] < keys[j][0]+keys[j][1] })
	for _, k := range keys {
		e := edges[k]
		p1, p2 := at[k[0]], at[k[1]]
		span := p2.col - p1.col
		if span < 0 {
			span = -span
		}
		var d string
		switch {
		case span > 1:
			// arc over the intermediate columns, from the top edge of each node
			x1, x2 := p1.x+colW/2, p2.x+colW/2
			lift := float64(min(p1.y, p2.y)) - 18 - 20*float64(span)
			d = fmt.Sprintf("M%d %d C%d %.0f %d %.0f %d %d", x1, p1.y, x1, lift, x2, lift, x2, p2.y)
		case p2.x == p1.x:
			x := p1.x + colW
			d = fmt.Sprintf("M%d %d C%d %d %d %d %d %d", x, p1.y+nodeH/2, x+40, p1.y+nodeH/2, x+40, p2.y+nodeH/2, x, p2.y+nodeH/2)
		case p2.x > p1.x:
			x1, x2 := p1.x+colW, p2.x
			mx := (x1 + x2) / 2
			d = fmt.Sprintf("M%d %d C%d %d %d %d %d %d", x1, p1.y+nodeH/2, mx, p1.y+nodeH/2, mx, p2.y+nodeH/2, x2, p2.y+nodeH/2)
		default:
			x1, x2 := p1.x, p2.x+colW
			mx := (x1 + x2) / 2
			d = fmt.Sprintf("M%d %d C%d %d %d %d %d %d", x1, p1.y+nodeH/2+6, mx, p1.y+nodeH/2+6, mx, p2.y+nodeH/2+6, x2, p2.y+nodeH/2+6)
		}
		cls := "e"
		if e.bad {
			cls = "e bad"
		}
		sw := 1 + math.Log2(float64(e.n))*0.6
		fmt.Fprintf(&b, `<path class="%s" stroke-width="%.1f" marker-end="url(#lar)" d="%s"><title>%s → %s (%d)</title></path>`,
			cls, sw, d, esc(k[0]), esc(k[1]), e.n)
	}
	for _, l := range cols {
		for _, s := range byLayer[l] {
			p := at[s.Package]
			label := strings.TrimPrefix(strings.TrimPrefix(s.Package, prefix), ".")
			if label == "" {
				label = s.Package
			}
			fmt.Fprintf(&b, `<g><title>%s&#10;%d types · Ca %d · Ce %d · I %.2f</title><rect class="node L-%s" x="%d" y="%d" width="%d" height="%d" rx="6"/>`+
				`<text class="nt" x="%d" y="%d">%s</text><text class="ns" x="%d" y="%d">%d · Ca %d · Ce %d</text></g>`,
				esc(s.Package), s.Types, s.Afferent, s.Efferent, s.Instability, l, p.x, p.y, colW, nodeH,
				p.x+8, p.y+15, esc(trunc(label, 34)), p.x+8, p.y+29, s.Types, s.Afferent, s.Efferent)
		}
	}
	b.WriteString(`</svg>`)
	return b.String(), prefix
}

type ctxBox struct {
	title string
	items []string
	seen  map[string]bool
	n     int
}

// ContextSVG draws the system with its inbound, outbound, data and platform sides.
func ContextSVG(f *model.Facts, lang string) string {
	groups := map[string]*ctxBox{}
	order := map[string][]string{"in": {}, "out": {}, "data": {}, "plat": {}}
	add := func(side, title, item string) {
		k := side + "|" + title
		g := groups[k]
		if g == nil {
			g = &ctxBox{title: title, seen: map[string]bool{}}
			groups[k] = g
			order[side] = append(order[side], k)
		}
		if g.seen[item] {
			return
		}
		g.seen[item] = true
		g.n++
		if len(g.items) < 4 && !containsStr(g.items, item) {
			g.items = append(g.items, item)
		}
	}
	exposed := map[string]bool{}
	for _, c := range f.Contracts {
		if c.Direction == "exposes" {
			exposed[c.Kind+"|"+c.Match] = true
		}
	}
	for _, c := range f.Contracts {
		label := strings.ToUpper(c.Kind)
		if c.Direction == "calls" && exposed[c.Kind+"|"+c.Match] {
			continue // the system calls itself (e.g. CICS RETURN TRANSID)
		}
		switch {
		case c.Kind == "db-table":
			add("data", tr(lang, "tables"), c.Match)
		case c.Kind == "dataset" || c.Kind == "cics-file":
			add("data", strings.ToUpper(c.Kind), c.Key)
		case c.Kind == "cics-queue":
			add("out", "CICS QUEUE", c.Key)
		case c.Direction == "exposes":
			add("in", label, c.Key)
		case c.Direction == "subscribes":
			add("in", label+" ⟵", c.Key)
		case c.Direction == "calls", c.Direction == "publishes":
			add("out", label, c.Key)
		case c.Kind == "jms" || c.Kind == "kafka":
			add("out", label, c.Key)
		case c.Kind == "jndi":
			add("out", "JNDI", c.Key)
		case c.Kind == "platform-module":
			add("plat", "JBoss modules", c.Key)
		}
	}
	seen := map[string]bool{}
	for _, fd := range f.Findings {
		if fd.SourceSet != "main" || seen[fd.Technology] {
			continue
		}
		switch fd.Category {
		case "Database":
			seen[fd.Technology] = true
			add("data", "DB", fd.Technology)
		case "Platform":
			seen[fd.Technology] = true
			add("plat", tr(lang, "platform"), fd.Technology)
		}
	}
	for _, m := range f.Modules {
		if m.Runtime != "" && !seen[m.Runtime] {
			seen[m.Runtime] = true
			add("plat", tr(lang, "runtime"), m.Runtime)
		}
	}
	const bw, bh = 250, 92
	rows := 1
	for _, s := range []string{"in", "out"} {
		if len(order[s]) > rows {
			rows = len(order[s])
		}
	}
	topH := 0
	if len(order["plat"]) > 0 {
		topH = bh + 40
	}
	midH := rows*(bh+14) + 10
	botH := 0
	if len(order["data"]) > 0 {
		botH = bh + 40
	}
	w := 3*bw + 2*110 + 40
	h := 30 + topH + midH + botH
	cx, cy := 20+bw+110, 30+topH+midH/2-40
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" role="img" aria-label="%s">`, w, h, w, esc(tr(lang, "context")))
	b.WriteString(`<defs><marker id="ar" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="#8a96a6"/></marker></defs>`)
	box := func(x, y int, g *ctxBox) {
		fmt.Fprintf(&b, `<g><rect class="box" x="%d" y="%d" width="%d" height="%d" rx="8"/><text class="bt" x="%d" y="%d">%s (%d)</text>`,
			x, y, bw, bh, x+10, y+18, esc(g.title), g.n)
		for i, it := range g.items {
			fmt.Fprintf(&b, `<text class="bi" x="%d" y="%d">%s</text>`, x+10, y+34+i*14, esc(trunc(it, 42)))
		}
		b.WriteString(`</g>`)
	}
	line := func(x1, y1, x2, y2 int) {
		fmt.Fprintf(&b, `<path class="e" stroke-width="1.4" marker-end="url(#ar)" d="M%d %d L%d %d"/>`, x1, y1, x2, y2)
	}
	fmt.Fprintf(&b, `<rect class="sys" x="%d" y="%d" width="%d" height="80" rx="10"/><text class="syst" x="%d" y="%d" text-anchor="middle">%s</text>`,
		cx, cy, bw, cx+bw/2, cy+45, esc(trunc(f.Repo.Name, 30)))
	for i, k := range order["in"] {
		y := 30 + topH + i*(bh+14)
		box(20, y, groups[k])
		line(20+bw, y+bh/2, cx, cy+40)
	}
	for i, k := range order["out"] {
		y := 30 + topH + i*(bh+14)
		x := cx + bw + 110
		box(x, y, groups[k])
		line(cx+bw, cy+40, x, y+bh/2)
	}
	for i, k := range order["plat"] {
		x := 20 + i*(bw+20)
		if i > 2 {
			break
		}
		box(x, 20, groups[k])
	}
	for i, k := range order["data"] {
		if i > 2 {
			break
		}
		x := cx - (len(order["data"])-1)*(bw+20)/2 + i*(bw+20)
		y := 30 + topH + midH + 20
		box(x, y, groups[k])
		line(cx+bw/2, cy+80, x+bw/2, y)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// LandscapeSVG draws systems on a circle with typed relationships.
func LandscapeSVG(p *model.Portfolio, lang string) string {
	n := len(p.Systems)
	if n == 0 {
		return ""
	}
	r := math.Max(170, float64(n)*30)
	const bw, bh = 168, 46
	w := int(2*r) + bw + 60
	if w < 760 {
		w = 760
	}
	h := int(2*r) + bh + 90
	cx, cy := float64(w)/2, float64(h-50)/2+10
	pos := map[string][2]float64{}
	for i, s := range p.Systems {
		a := 2*math.Pi*float64(i)/float64(n) - math.Pi/2
		if n == 1 {
			pos[s.Name] = [2]float64{cx, cy}
			continue
		}
		pos[s.Name] = [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)}
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" role="img" aria-label="%s">`, w, h, w, esc(tr(lang, "landscape")))
	b.WriteString(`<defs><marker id="la" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10z" fill="#8a96a6"/></marker></defs>`)
	pairIdx := map[string]int{}
	for _, l := range p.Links {
		a, ok1 := pos[l.From]
		c, ok2 := pos[l.To]
		if !ok1 || !ok2 {
			continue
		}
		pk := l.From + "|" + l.To
		if l.To < l.From {
			pk = l.To + "|" + l.From
		}
		k := pairIdx[pk]
		pairIdx[pk]++
		mx, my := (a[0]+c[0])/2, (a[1]+c[1])/2
		// bend toward the centre, more for parallel links
		bend := 0.15
		qx, qy := mx+(cx-mx)*bend, my+(cy-my)*bend
		if k > 0 { // separate parallel links perpendicular to the chord
			dx, dy := c[0]-a[0], c[1]-a[1]
			ln := math.Hypot(dx, dy)
			off := 28 * float64(k)
			if k%2 == 0 {
				off = -off
			}
			qx, qy = qx-dy/ln*off, qy+dx/ln*off
		}
		sx, sy := clipToBox(a[0], a[1], qx, qy, bw, bh)
		ex, ey := clipToBox(c[0], c[1], qx, qy, bw, bh)
		lo := ""
		if l.Confidence == "low" {
			lo = " lo"
		}
		fmt.Fprintf(&b, `<path class="lk k-%s%s" d="M%.0f %.0f Q%.0f %.0f %.0f %.0f" marker-end="url(#la)"><title>%s → %s · %s (%d) · confidence %s&#10;%s</title></path>`,
			esc(l.Kind), lo, sx, sy, qx, qy, ex, ey, esc(l.From), esc(l.To), esc(l.Kind), l.Count, esc(l.Confidence), esc(strings.Join(l.Keys, "\n")))
	}
	for _, s := range p.Systems {
		q := pos[s.Name]
		x, y := q[0]-bw/2, q[1]-bh/2
		loc := fmt.Sprintf("%d LOC", s.Code)
		if s.Code >= 1000 {
			loc = fmt.Sprintf("%.1fk LOC", float64(s.Code)/1000)
		}
		sub := strings.Join(firstStrs(s.Languages, 2), "/") + " · " + loc
		fmt.Fprintf(&b, `<g><title>%s</title><rect class="sys" x="%.0f" y="%.0f" width="%d" height="%d" rx="8"/><text class="syst" x="%.0f" y="%.0f" text-anchor="middle">%s</text>`+
			`<text class="syst" style="font-size:10px;font-weight:400" x="%.0f" y="%.0f" text-anchor="middle">%s</text></g>`,
			esc(s.Name), x, y, bw, bh, q[0], y+19, esc(trunc(s.Name, 22)), q[0], y+35, esc(trunc(sub, 28)))
	}
	kinds := []string{"library", "http", "soap", "jms", "ejb", "shared-db", "deploys"}
	for i, k := range kinds {
		x := 20 + i*(w-40)/len(kinds)
		fmt.Fprintf(&b, `<path class="lk k-%s" d="M%d %d h28"/><text class="lg" x="%d" y="%d">%s</text>`, k, x, h-22, x+34, h-18, k)
	}
	fmt.Fprintf(&b, `<path class="lk lo" style="stroke:#8a96a6" d="M%d %d h28"/><text class="lg" x="%d" y="%d">%s</text>`, 20, h-4, 54, h, esc(tr(lang, "lowConfidence")))
	b.WriteString(`</svg>`)
	return b.String()
}

// clipToBox moves a box centre (x,y) towards (tx,ty) until it reaches the box border.
func clipToBox(x, y, tx, ty float64, bw, bh int) (float64, float64) {
	dx, dy := tx-x, ty-y
	if dx == 0 && dy == 0 {
		return x, y
	}
	sx, sy := math.Inf(1), math.Inf(1)
	if dx != 0 {
		sx = (float64(bw)/2 + 3) / math.Abs(dx)
	}
	if dy != 0 {
		sy = (float64(bh)/2 + 3) / math.Abs(dy)
	}
	k := math.Min(math.Min(sx, sy), 1)
	return x + dx*k, y + dy*k
}

func firstStrs(xs []string, n int) []string {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// Standalone wraps an inline SVG with a light theme for use outside the report.
func Standalone(svg string) string {
	if svg == "" {
		return ""
	}
	i := strings.Index(svg, ">")
	return svg[:i+1] + svgStandaloneStyle + `<rect width="100%" height="100%" fill="#fff"/>` + svg[i+1:]
}
