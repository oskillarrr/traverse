// Package deploy reads infrastructure-as-code: Dockerfiles, Docker Compose,
// Kubernetes, OpenShift and Helm manifests. Only names, images, ports,
// replicas, schedules and environment variable NAMES are kept; values are
// inspected only to discover references to other services and are never
// stored. Host names (registries, ingress hosts) are returned separately so
// the scanner can seal them.
package deploy

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/yamlite"
)

// Result is what one file contributes.
type Result struct {
	Nodes    []model.DeployNode
	Links    []model.DeployLink
	Contexts map[string]string // node ID -> build context directory (repo-relative)
	Findings []Finding
	Err      error
}

type Finding struct {
	RuleID, Title, Category, Technology, Severity, Message string
	Line                                                   int
}

// Detect reports which kind of IaC file a path/content is ("" if none).
func Detect(rel, language, text string) string {
	base := strings.ToLower(path.Base(rel))
	switch {
	case language == "Dockerfile":
		return "dockerfile"
	case strings.HasPrefix(base, "docker-compose") || strings.HasPrefix(base, "compose.") || base == "compose.yml" || base == "compose.yaml":
		return "compose"
	case language == "YAML":
		head := text
		if len(head) > 20000 {
			head = head[:20000]
		}
		if strings.Contains(head, "apiVersion:") && strings.Contains(head, "kind:") {
			if strings.Contains(rel, "/templates/") {
				return "helm"
			}
			return "kubernetes"
		}
		if strings.Contains(head, "services:") && (strings.Contains(head, "image:") || strings.Contains(head, "build:")) {
			return "compose"
		}
	}
	return ""
}

// Parse extracts deployment nodes from one file.
func Parse(rel, kind, text string) *Result {
	r := &Result{Contexts: map[string]string{}}
	switch kind {
	case "dockerfile":
		dockerfile(r, rel, text)
	case "compose":
		docs, err := yamlite.ParseAll(text)
		r.Err = err
		for _, d := range docs {
			compose(r, rel, d)
		}
	case "kubernetes", "helm":
		src := text
		platform := "kubernetes"
		if kind == "helm" {
			src, platform = yamlite.HelmSafe(text), "helm"
		}
		docs, err := yamlite.ParseAll(src)
		r.Err = err
		for _, d := range docs {
			k8s(r, rel, platform, d)
		}
	}
	return r
}

// SplitImage removes the registry host from an image reference.
func SplitImage(ref string) (host, image string) {
	ref = strings.TrimSpace(ref)
	if i := strings.IndexByte(ref, '/'); i > 0 {
		first := ref[:i]
		if strings.ContainsAny(first, ".:") || first == "localhost" {
			return strings.Split(first, ":")[0], ref[i+1:]
		}
	}
	return "", ref
}

// ImageName returns the repository's last path segment without tag/digest.
func ImageName(image string) string {
	_, img := SplitImage(image)
	img = strings.SplitN(img, "@", 2)[0]
	if i := strings.LastIndexByte(img, '/'); i >= 0 {
		img = img[i+1:]
	}
	if i := strings.IndexByte(img, ':'); i >= 0 {
		img = img[:i]
	}
	return img
}

var (
	reFrom     = regexp.MustCompile(`(?im)^\s*FROM\s+(?:--platform=\S+\s+)?(\S+)(?:\s+AS\s+(\S+))?`)
	reExpose   = regexp.MustCompile(`(?im)^\s*EXPOSE\s+(.+)$`)
	reUser     = regexp.MustCompile(`(?im)^\s*USER\s+(\S+)`)
	reEnvKey   = regexp.MustCompile(`(?im)^\s*(?:ENV|ARG)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	reAddURL   = regexp.MustCompile(`(?im)^\s*ADD\s+https?://`)
	reHealth   = regexp.MustCompile(`(?im)^\s*HEALTHCHECK\s`)
	reEOLImage = regexp.MustCompile(`(?i)^(?:[\w.-]+/)*(?:openjdk|java|eclipse-temurin|adoptopenjdk):(?:6|7|8)\b|^(?:jboss/)?wildfly:(?:[0-9]|1[0-9])\.|^tomcat:(?:[5-8])(?:\.|-|$)|^node:(?:[0-9]|1[0-7])(?:\.|-|$)|^python:2\b|^php:(?:[5]|7)\.|^centos:|^ubuntu:(?:1[0-9])\.|^debian:(?:[0-9]|10)(?:\.|-|$)|^mcr\.microsoft\.com/dotnet/(?:core/)?(?:aspnet|runtime|sdk):(?:[1-5])\.`)
	reSvcRef   = regexp.MustCompile(`(?:^|[/@=,\s])([a-z0-9][a-z0-9_.-]*)(?::\d{2,5})`)
)

func dockerfile(r *Result, rel, text string) {
	dir := path.Dir(rel)
	if dir == "." {
		dir = ""
	}
	name := "root"
	if dir != "" {
		name = path.Base(dir)
	}
	n := model.DeployNode{ID: "docker:" + rel, Platform: "docker", Kind: "image", Name: name, File: rel, Line: 1}
	stages := map[string]bool{}
	var lastBase string
	for _, m := range reFrom.FindAllStringSubmatchIndex(text, -1) {
		img := text[m[2]:m[3]]
		if m[4] >= 0 {
			stages[strings.ToLower(text[m[4]:m[5]])] = true
		}
		if stages[strings.ToLower(img)] {
			continue // FROM a previous build stage
		}
		host, clean := SplitImage(img)
		if host != "" {
			n.Hosts = append(n.Hosts, host)
		}
		n.Base = append(n.Base, clean)
		lastBase = clean
		line := strings.Count(text[:m[0]], "\n") + 1
		if !strings.Contains(clean, ":") && !strings.Contains(clean, "@") || strings.HasSuffix(clean, ":latest") {
			r.Findings = append(r.Findings, Finding{"OPS-DOCKER-UNPINNED", "Base image without a pinned version", "Delivery", "Docker", "medium",
				"Builds are not reproducible and may change without notice.", line})
		}
		if reEOLImage.MatchString(clean) {
			r.Findings = append(r.Findings, Finding{"OPS-DOCKER-EOL-BASE", "End-of-life base image: " + clean, "Migration", "Docker", "high",
				"The base image no longer receives security updates.", line})
		}
	}
	for _, m := range reExpose.FindAllStringSubmatch(text, -1) {
		n.Ports = append(n.Ports, strings.Fields(m[1])...)
	}
	if m := reUser.FindAllStringSubmatch(text, -1); len(m) > 0 {
		n.User = m[len(m)-1][1]
	}
	if n.User == "" || n.User == "root" || n.User == "0" {
		r.Findings = append(r.Findings, Finding{"OPS-DOCKER-ROOT", "Container runs as root", "Security", "Docker", "medium",
			"No USER instruction (or USER root): a container escape gives root privileges.", 1})
	}
	if reAddURL.MatchString(text) {
		r.Findings = append(r.Findings, Finding{"OPS-DOCKER-ADD-URL", "ADD downloads remote content at build time", "Security", "Docker", "low",
			"Prefer verified, pinned downloads (curl + checksum) or COPY.", 1})
	}
	if !reHealth.MatchString(text) {
		r.Findings = append(r.Findings, Finding{"OPS-DOCKER-NO-HEALTHCHECK", "Image without HEALTHCHECK", "Delivery", "Docker", "info", "", 1})
	}
	for _, m := range reEnvKey.FindAllStringSubmatch(text, -1) {
		n.EnvKeys = appendUnique(n.EnvKeys, m[1])
	}
	_ = lastBase
	r.Contexts[n.ID] = dir
	r.Nodes = append(r.Nodes, n)
}

func compose(r *Result, rel string, d *yamlite.Node) {
	svcs := d.Get("services")
	if svcs == nil {
		return
	}
	dir := path.Dir(rel)
	names := svcs.Keys()
	known := map[string]bool{}
	for _, s := range names {
		known[s] = true
	}
	id := func(s string) string { return "compose:" + rel + "#" + s }
	for _, p := range svcs.Entries() {
		s := p.Value
		if s == nil {
			continue
		}
		n := model.DeployNode{ID: id(p.Key), Platform: "compose", Kind: "service", Name: p.Key, File: rel, Line: s.Line}
		if img := s.S("image"); img != "" {
			host, clean := SplitImage(img)
			n.Images = []string{clean}
			if host != "" {
				n.Hosts = append(n.Hosts, host)
			}
		}
		if b := s.Get("build"); b != nil {
			ctx := b.Str()
			if b.Kind == yamlite.Map {
				ctx = b.S("context")
			}
			if ctx != "" && !strings.Contains(ctx, "://") {
				r.Contexts[n.ID] = path.Clean(path.Join(dir, ctx))
			}
		}
		for _, port := range s.Get("ports").List() {
			v := port.Str()
			if port != nil && port.Kind == yamlite.Map {
				v = port.S("published") + ":" + port.S("target")
			}
			n.Ports = append(n.Ports, v)
		}
		if rep := s.S("deploy", "replicas"); rep != "" {
			n.Replicas = rep
		}
		deps := s.Get("depends_on")
		for _, x := range deps.List() {
			r.Links = append(r.Links, model.DeployLink{From: n.ID, To: id(x.Str()), Kind: "depends_on"})
		}
		for _, k := range deps.Keys() {
			r.Links = append(r.Links, model.DeployLink{From: n.ID, To: id(k), Kind: "depends_on"})
		}
		for _, x := range s.Get("links").List() {
			t := strings.SplitN(x.Str(), ":", 2)[0]
			r.Links = append(r.Links, model.DeployLink{From: n.ID, To: id(t), Kind: "depends_on"})
		}
		// environment: keep names; inspect values only for references to other services
		env := s.Get("environment")
		var values []string
		for _, x := range env.List() {
			k, v, _ := strings.Cut(x.Str(), "=")
			n.EnvKeys = appendUnique(n.EnvKeys, k)
			values = append(values, v)
		}
		for _, pr := range env.Entries() {
			n.EnvKeys = appendUnique(n.EnvKeys, pr.Key)
			values = append(values, pr.Value.Str())
		}
		for _, v := range values {
			for _, ref := range serviceRefs(v, known) {
				if ref != p.Key {
					r.Links = append(r.Links, model.DeployLink{From: n.ID, To: id(ref), Kind: "env-ref"})
				}
			}
		}
		sort.Strings(n.EnvKeys)
		r.Nodes = append(r.Nodes, n)
	}
	r.Links = dedupeLinks(r.Links)
}

func serviceRefs(v string, known map[string]bool) []string {
	var out []string
	for _, m := range reSvcRef.FindAllStringSubmatch(v, -1) {
		if known[m[1]] {
			out = append(out, m[1])
		}
	}
	// "http://svc/path" without port
	for name := range known {
		if strings.Contains(v, "://"+name+"/") || strings.HasSuffix(v, "://"+name) {
			out = append(out, name)
		}
	}
	return out
}

var workloads = map[string]bool{"Deployment": true, "StatefulSet": true, "DaemonSet": true, "ReplicaSet": true, "Job": true,
	"CronJob": true, "Pod": true, "DeploymentConfig": true, "Rollout": true, "KnativeService": true}

func k8s(r *Result, rel, platform string, d *yamlite.Node) {
	kind := d.S("kind")
	name := d.S("metadata", "name")
	if kind == "" || name == "" {
		return
	}
	if kind == "List" {
		for _, it := range d.Get("items").List() {
			k8s(r, rel, platform, it)
		}
		return
	}
	api := d.S("apiVersion")
	if strings.Contains(api, "openshift.io") {
		platform = "openshift"
	}
	ns := d.S("metadata", "namespace")
	n := model.DeployNode{ID: platform + ":" + ns + "/" + kind + "/" + name + "@" + rel, Platform: platform, Kind: kind, Name: name,
		Namespace: ns, File: rel, Line: d.Line}
	if kind == "Service" && strings.HasPrefix(api, "serving.knative") {
		n.Kind = "KnativeService"
	}
	spec := d.Get("spec")
	if workloads[n.Kind] {
		n.Replicas = spec.S("replicas")
		pod := spec.Path("template", "spec")
		labels := spec.Path("template", "metadata", "labels")
		if kind == "CronJob" {
			n.Schedule = spec.S("schedule")
			pod = spec.Path("jobTemplate", "spec", "template", "spec")
			labels = spec.Path("jobTemplate", "spec", "template", "metadata", "labels")
		}
		if kind == "Pod" {
			pod, labels = spec, d.Path("metadata", "labels")
		}
		for _, p := range labels.Entries() {
			n.Labels = append(n.Labels, p.Key+"="+p.Value.Str())
		}
		for _, list := range []string{"initContainers", "containers"} {
			for _, c := range pod.Get(list).List() {
				if img := c.S("image"); img != "" {
					host, clean := SplitImage(img)
					n.Images = appendUnique(n.Images, clean)
					if host != "" {
						n.Hosts = appendUnique(n.Hosts, host)
					}
				}
				for _, p := range c.Get("ports").List() {
					n.Ports = appendUnique(n.Ports, p.S("containerPort"))
				}
				for _, e := range c.Get("env").List() {
					n.EnvKeys = appendUnique(n.EnvKeys, e.S("name"))
					if cm := e.S("valueFrom", "configMapKeyRef", "name"); cm != "" {
						r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:ConfigMap/" + cm, Kind: "config"})
					}
					if sec := e.S("valueFrom", "secretKeyRef", "name"); sec != "" {
						r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:Secret/" + sec, Kind: "config"})
					}
				}
				for _, ef := range c.Get("envFrom").List() {
					if cm := ef.S("configMapRef", "name"); cm != "" {
						r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:ConfigMap/" + cm, Kind: "config"})
					}
					if sec := ef.S("secretRef", "name"); sec != "" {
						r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:Secret/" + sec, Kind: "config"})
					}
				}
				if c.S("securityContext", "privileged") == "true" {
					r.Findings = append(r.Findings, Finding{"OPS-K8S-PRIVILEGED", "Privileged container", "Security", "Kubernetes", "high",
						"Privileged containers have full access to the node.", c.Line})
				}
				if c.Get("resources") == nil {
					r.Findings = append(r.Findings, Finding{"OPS-K8S-NO-RESOURCES", "Container without resource requests/limits", "Delivery", "Kubernetes", "low",
						"Without limits a container can starve the node.", c.Line})
				}
				if strings.HasSuffix(c.S("image"), ":latest") || (c.S("image") != "" && !strings.Contains(path.Base(c.S("image")), ":") && !strings.Contains(c.S("image"), "@")) {
					r.Findings = append(r.Findings, Finding{"OPS-K8S-UNPINNED", "Image without a pinned version", "Delivery", "Kubernetes", "medium", "", c.Line})
				}
			}
		}
		sort.Strings(n.EnvKeys)
	}
	switch kind {
	case "Service":
		for _, p := range spec.Get("ports").List() {
			n.Ports = appendUnique(n.Ports, strings.Trim(p.S("port")+"→"+p.S("targetPort"), "→"))
		}
		if t := spec.S("type"); t != "" {
			n.Kind = "Service (" + t + ")"
		}
		for _, p := range spec.Get("selector").Entries() {
			r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "selector:" + p.Key + "=" + p.Value.Str(), Kind: "selects"})
		}
	case "Ingress":
		for _, rule := range spec.Get("rules").List() {
			if h := rule.S("host"); h != "" {
				n.Hosts = appendUnique(n.Hosts, h)
			}
			for _, p := range rule.Path("http", "paths").List() {
				n.Paths = appendUnique(n.Paths, firstNonEmpty(p.S("path"), "/"))
				svc := firstNonEmpty(p.S("backend", "service", "name"), p.S("backend", "serviceName"))
				if svc != "" {
					r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:Service/" + svc, Kind: "routes"})
				}
			}
		}
	case "Route":
		if h := spec.S("host"); h != "" {
			n.Hosts = appendUnique(n.Hosts, h)
		}
		n.Paths = appendUnique(n.Paths, firstNonEmpty(spec.S("path"), "/"))
		if svc := spec.S("to", "name"); svc != "" {
			r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:Service/" + svc, Kind: "routes"})
		}
	case "ConfigMap":
		for _, k := range d.Get("data").Keys() {
			n.EnvKeys = append(n.EnvKeys, k)
		}
	case "Secret":
		for _, k := range append(d.Get("data").Keys(), d.Get("stringData").Keys()...) {
			n.EnvKeys = append(n.EnvKeys, k)
		}
		if len(n.EnvKeys) > 0 {
			r.Findings = append(r.Findings, Finding{"OPS-K8S-SECRET-IN-REPO", "Kubernetes Secret with data in the repository", "Security", "Kubernetes", "high",
				"Secret values are only base64-encoded; use a secret manager or sealed secrets. Values are never copied to the report.", d.Line})
		}
	case "HorizontalPodAutoscaler":
		if t := spec.S("scaleTargetRef", "name"); t != "" {
			r.Links = append(r.Links, model.DeployLink{From: n.ID, To: "ref:" + spec.S("scaleTargetRef", "kind") + "/" + t, Kind: "scales"})
		}
		n.Replicas = strings.Trim(spec.S("minReplicas")+"-"+spec.S("maxReplicas"), "-")
	}
	r.Nodes = append(r.Nodes, n)
}

// Resolve turns symbolic link targets (ref:Kind/name, selector:k=v) into node
// IDs once every file of the repository has been read.
func Resolve(nodes []model.DeployNode, links []model.DeployLink) []model.DeployLink {
	byKindName := map[string]string{}
	for _, n := range nodes {
		base := strings.SplitN(n.Kind, " ", 2)[0]
		byKindName[base+"/"+n.Name] = n.ID
		if workloads[base] {
			byKindName["workload/"+n.Name] = n.ID
		}
	}
	var out []model.DeployLink
	for _, l := range links {
		switch {
		case strings.HasPrefix(l.To, "ref:"):
			t := strings.TrimPrefix(l.To, "ref:")
			id, ok := byKindName[t]
			if !ok {
				parts := strings.SplitN(t, "/", 2)
				id, ok = byKindName["workload/"+parts[len(parts)-1]]
			}
			if ok {
				out = append(out, model.DeployLink{From: l.From, To: id, Kind: l.Kind})
			}
		case strings.HasPrefix(l.To, "selector:"):
			sel := strings.TrimPrefix(l.To, "selector:")
			for _, n := range nodes {
				for _, lb := range n.Labels {
					if lb == sel {
						out = append(out, model.DeployLink{From: l.From, To: n.ID, Kind: l.Kind})
					}
				}
			}
		default:
			out = append(out, l)
		}
	}
	return dedupeLinks(out)
}

func dedupeLinks(ls []model.DeployLink) []model.DeployLink {
	seen := map[string]bool{}
	var out []model.DeployLink
	for _, l := range ls {
		k := l.From + "|" + l.To + "|" + l.Kind
		if !seen[k] && l.From != l.To {
			seen[k] = true
			out = append(out, l)
		}
	}
	return out
}

func appendUnique(xs []string, v string) []string {
	if v == "" {
		return xs
	}
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
