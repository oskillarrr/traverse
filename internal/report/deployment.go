package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// deployColumn groups deployment node kinds for the diagram.
func deployColumn(n model.DeployNode) int {
	k := strings.SplitN(n.Kind, " ", 2)[0]
	switch {
	case k == "image":
		return 0
	case k == "service" && n.Platform == "compose":
		return 1
	case k == "Ingress" || k == "Route":
		return 2
	case k == "Service":
		return 3
	case k == "ConfigMap" || k == "Secret" || k == "HorizontalPodAutoscaler":
		return 5
	}
	return 4 // workloads
}

var deployColumns = []string{"deployImages", "deployCompose", "deployIngress", "deployServices", "deployWorkloads", "deployConfig"}

// DeploymentSVG draws images, Compose services and Kubernetes/OpenShift
// objects in columns with their relationships.
func DeploymentSVG(f *model.Facts, lang string) string {
	nodes := f.Deployment.Nodes
	if len(nodes) == 0 {
		return ""
	}
	if len(nodes) > 120 {
		nodes = nodes[:120]
	}
	byCol := map[int][]model.DeployNode{}
	for _, n := range nodes {
		c := deployColumn(n)
		byCol[c] = append(byCol[c], n)
	}
	var cols []int
	for c := range deployColumns {
		if len(byCol[c]) > 0 {
			cols = append(cols, c)
		}
	}
	const colW, gap, nodeH, vgap, top = 210, 60, 44, 10, 46
	type pos struct{ x, y int }
	at := map[string]pos{}
	maxRows := 0
	for i, c := range cols {
		sort.Slice(byCol[c], func(a, b int) bool { return byCol[c][a].Name < byCol[c][b].Name })
		for j, n := range byCol[c] {
			at[n.ID] = pos{20 + i*(colW+gap), top + j*(nodeH+vgap)}
		}
		if len(byCol[c]) > maxRows {
			maxRows = len(byCol[c])
		}
	}
	w := 40 + len(cols)*(colW+gap) - gap
	h := top + maxRows*(nodeH+vgap) + 20
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" role="img" aria-label="%s">`, w, h, w, esc(tr(lang, "deployment")))
	b.WriteString(`<defs><marker id="da" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0L10 5L0 10z" fill="#8a96a6"/></marker></defs>`)
	for i, c := range cols {
		x := 20 + i*(colW+gap)
		fmt.Fprintf(&b, `<rect class="col" x="%d" y="10" width="%d" height="%d" rx="8"/><text class="colt" x="%d" y="30">%s</text>`, x-8, colW+16, h-20, x, esc(tr(lang, deployColumns[c])))
	}
	for _, l := range f.Deployment.Links {
		a, ok1 := at[l.From]
		c, ok2 := at[l.To]
		if !ok1 || !ok2 {
			continue
		}
		x1, y1, x2, y2 := a.x+colW, a.y+nodeH/2, c.x, c.y+nodeH/2
		if c.x < a.x {
			x1, x2 = a.x, c.x+colW
		} else if c.x == a.x {
			x1, x2 = a.x+colW, c.x+colW
		}
		mx := (x1 + x2) / 2
		if c.x == a.x {
			mx = x1 + 40
		}
		fmt.Fprintf(&b, `<path class="e" stroke-width="1.3" marker-end="url(#da)" d="M%d %d C%d %d %d %d %d %d"><title>%s</title></path>`, x1, y1, mx, y1, mx, y2, x2, y2, esc(l.Kind))
	}
	for _, n := range nodes {
		p := at[n.ID]
		sub := strings.Join(firstStrs(n.Images, 1), "")
		if sub == "" {
			sub = strings.Join(firstStrs(n.Base, 1), "")
		}
		if n.Replicas != "" {
			sub = "×" + n.Replicas + " " + sub
		}
		if n.Schedule != "" {
			sub = n.Schedule
		}
		cls := "L-Configuration"
		switch deployColumn(n) {
		case 0, 1, 4:
			cls = "L-Business"
		case 2, 3:
			cls = "L-Presentation"
		}
		fmt.Fprintf(&b, `<g><title>%s · %s · %s</title><rect class="node %s" x="%d" y="%d" width="%d" height="%d" rx="6"/>`+
			`<text class="nt" x="%d" y="%d">%s</text><text class="ns" x="%d" y="%d">%s</text></g>`,
			esc(n.Platform), esc(n.Kind), esc(n.File), cls, p.x, p.y, colW, nodeH,
			p.x+8, p.y+17, esc(trunc(n.Name+" · "+n.Kind, 32)), p.x+8, p.y+33, esc(trunc(sub, 34)))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// structurizrDeployment renders C4 deployment environments (one per platform).
func structurizrDeployment(f *model.Facts, sys string, containerOf map[string]string) (string, []string) {
	byPlat := map[string][]model.DeployNode{}
	var plats []string
	for _, n := range f.Deployment.Nodes {
		if _, ok := byPlat[n.Platform]; !ok {
			plats = append(plats, n.Platform)
		}
		byPlat[n.Platform] = append(byPlat[n.Platform], n)
	}
	sort.Strings(plats)
	var b strings.Builder
	var views []string
	known := map[string]bool{}
	for _, p := range plats {
		env := map[string]string{"docker": "Docker images", "compose": "Docker Compose", "kubernetes": "Kubernetes", "openshift": "OpenShift", "helm": "Helm"}[p]
		if env == "" {
			env = p
		}
		cluster := map[string]string{"docker": "Container host", "compose": "Docker Compose host", "kubernetes": "Kubernetes cluster",
			"openshift": "OpenShift cluster", "helm": "Kubernetes cluster (Helm)"}[p]
		fmt.Fprintf(&b, "    deploymentEnvironment \"%s\" {\n      %s = deploymentNode \"%s\" \"Declared in the repository\" \"%s\" {\n", q(env), ident("cl_", p), q(firstNonEmpty(cluster, env)), q(p))
		names := map[string]int{}
		for _, n := range byPlat[p] {
			id := ident("d_", n.ID)
			name := uniq(n)
			names[name]++
			if names[name] > 1 { // e.g. docker-compose.override.yml redefines a service
				name += " [" + n.File + "]"
			}
			known[n.ID] = true
			tech := strings.TrimSpace(n.Kind + " " + strings.Join(n.Images, " ") + " " + strings.Join(n.Base, " "))
			inst := "1"
			if n.Replicas != "" && strings.Trim(n.Replicas, "0123456789") == "" {
				inst = n.Replicas
			}
			switch deployColumn(n) {
			case 2, 3, 5:
				fmt.Fprintf(&b, "        %s = infrastructureNode \"%s\" \"%s\" \"%s\"\n", id, q(name), q(n.File), q(tech))
			default:
				fmt.Fprintf(&b, "        %s = deploymentNode \"%s\" \"%s\" \"%s\" %s {\n", id, q(name), q(n.File), q(tech), inst)
				if c, ok := containerOf[n.Module]; ok && n.Module != "" {
					fmt.Fprintf(&b, "          containerInstance %s\n", c)
				}
				b.WriteString("        }\n")
			}
		}
		b.WriteString("      }\n")
		for _, l := range f.Deployment.Links {
			if known[l.From] && known[l.To] && platformOf(f, l.From) == p && platformOf(f, l.To) == p {
				fmt.Fprintf(&b, "      %s -> %s \"%s\"\n", ident("d_", l.From), ident("d_", l.To), q(l.Kind))
			}
		}
		b.WriteString("    }\n")
		views = append(views, fmt.Sprintf("    deployment %s \"%s\" \"%s\" {\n      include *\n      autoLayout lr\n    }\n", sys, q(env), ident("Deployment_", p)))
	}
	return b.String(), views
}

// uniq names a node uniquely within its cluster (Kubernetes objects of
// different kinds commonly share a name), as C4 requires.
func uniq(n model.DeployNode) string {
	name := n.Name + " (" + n.Kind + ")"
	if n.Namespace != "" {
		name = n.Namespace + "/" + name
	}
	return name
}

func platformOf(f *model.Facts, id string) string {
	for _, n := range f.Deployment.Nodes {
		if n.ID == id {
			return n.Platform
		}
	}
	return ""
}
