package scan

import (
	"sort"

	"github.com/oskillarrr/traverse/internal/model"
)

// metrics computes package coupling, cycles and layer violations on main code.
func metrics(f *model.Facts) {
	comp := map[string]model.Component{}
	for _, c := range f.Components {
		comp[c.FQN] = c
	}
	type pk struct {
		types   int
		layer   map[string]int
		module  string
		in, out map[string]bool
	}
	pkgs := map[string]*pk{}
	get := func(p string) *pk {
		if x, ok := pkgs[p]; ok {
			return x
		}
		x := &pk{layer: map[string]int{}, in: map[string]bool{}, out: map[string]bool{}}
		pkgs[p] = x
		return x
	}
	for _, c := range f.Components {
		if c.SourceSet != "main" || c.Kind[0] == 'n' { // skip nested
			continue
		}
		p := get(c.Package)
		p.types++
		p.layer[c.Layer]++
		p.module = c.Module
	}
	graph := map[string]map[string]bool{}
	for _, d := range f.Dependencies {
		a, okA := comp[d.From]
		b, okB := comp[d.To]
		if !okA || !okB || a.SourceSet != "main" || b.SourceSet != "main" {
			continue
		}
		if a.Package != b.Package {
			get(a.Package).out[b.Package] = true
			get(b.Package).in[a.Package] = true
			if graph[a.Package] == nil {
				graph[a.Package] = map[string]bool{}
			}
			graph[a.Package][b.Package] = true
		}
		if Forbidden(a.Layer, b.Layer) {
			f.LayerIssues = append(f.LayerIssues, model.LayerIssue{FromLayer: a.Layer, ToLayer: b.Layer, From: a.FQN, To: b.FQN, File: d.File, Line: d.Line})
		}
	}
	for name, p := range pkgs {
		ca, ce := len(p.in), len(p.out)
		inst := 0.0
		if ca+ce > 0 {
			inst = float64(ce) / float64(ca+ce)
		}
		f.PackageStats = append(f.PackageStats, model.PackageStat{Package: name, Module: p.module, Layer: dominant(p.layer),
			Types: p.types, Afferent: ca, Efferent: ce, Instability: float64(int(inst*100+0.5)) / 100})
	}
	sort.Slice(f.PackageStats, func(i, j int) bool { return f.PackageStats[i].Package < f.PackageStats[j].Package })
	for _, scc := range SCC(graph) {
		f.Cycles = append(f.Cycles, model.Cycle{Level: "package", Members: scc})
	}
	// module-level cycles (internal module dependencies)
	ids := map[string]bool{}
	for _, m := range f.Modules {
		ids[m.ID] = true
	}
	mg := map[string]map[string]bool{}
	for _, m := range f.Modules {
		for _, d := range m.Deps {
			if ids[d.ID] && d.ID != m.ID && !d.Managed {
				if mg[m.ID] == nil {
					mg[m.ID] = map[string]bool{}
				}
				mg[m.ID][d.ID] = true
			}
		}
	}
	for _, scc := range SCC(mg) {
		f.Cycles = append(f.Cycles, model.Cycle{Level: "module", Members: scc})
	}
	sort.Slice(f.LayerIssues, func(i, j int) bool {
		a, b := f.LayerIssues[i], f.LayerIssues[j]
		if a.From != b.From {
			return a.From < b.From
		}
		return a.To < b.To
	})
}

func dominant(m map[string]int) string {
	best, n := LUnknown, -1
	for _, l := range LayerOrder {
		if m[l] > n && m[l] > 0 {
			best, n = l, m[l]
		}
	}
	return best
}

// SCC returns strongly connected components with more than one node (Tarjan).
func SCC(g map[string]map[string]bool) [][]string {
	var nodes []string
	seen := map[string]bool{}
	for a, outs := range g {
		if !seen[a] {
			seen[a] = true
			nodes = append(nodes, a)
		}
		for b := range outs {
			if !seen[b] {
				seen[b] = true
				nodes = append(nodes, b)
			}
		}
	}
	sort.Strings(nodes)
	idx, low := map[string]int{}, map[string]int{}
	on := map[string]bool{}
	var stack []string
	var out [][]string
	i := 0
	var strong func(v string)
	strong = func(v string) {
		idx[v], low[v] = i, i
		i++
		stack = append(stack, v)
		on[v] = true
		var succ []string
		for w := range g[v] {
			succ = append(succ, w)
		}
		sort.Strings(succ)
		for _, w := range succ {
			if _, ok := idx[w]; !ok {
				strong(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if on[w] && idx[w] < low[v] {
				low[v] = idx[w]
			}
		}
		if low[v] == idx[v] {
			var c []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				on[w] = false
				c = append(c, w)
				if w == v {
					break
				}
			}
			if len(c) > 1 {
				sort.Strings(c)
				out = append(out, c)
			}
		}
	}
	for _, v := range nodes {
		if _, ok := idx[v]; !ok {
			strong(v)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a][0] < out[b][0] })
	return out
}
