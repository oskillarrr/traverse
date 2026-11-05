// Package model defines the facts schema: the stable contract between
// extraction, aggregation and reporting. Any extractor (built-in or external)
// that emits this schema can feed the rest of the pipeline.
package model

// SchemaVersion follows semver. Bump MINOR for additive changes, MAJOR for breaking ones.
const SchemaVersion = "1.5.0"

// Facts is everything extracted from a single repository.
type Facts struct {
	SchemaVersion string        `json:"schemaVersion"`
	Tool          ToolInfo      `json:"tool"`
	Repo          RepoInfo      `json:"repo"`
	Languages     []LangStat    `json:"languages"`
	Files         []File        `json:"files"`
	Modules       []Module      `json:"modules"`
	Deployables   []Deployable  `json:"deployables"`
	Components    []Component   `json:"components"`
	Dependencies  []CodeDep     `json:"dependencies"`
	Contracts     []Contract    `json:"contracts"`
	Findings      []Finding     `json:"findings"`
	PackageStats  []PackageStat `json:"packageStats"`
	Cycles        []Cycle       `json:"cycles"`
	LayerIssues   []LayerIssue  `json:"layerIssues"`
	Deployment    Deployment    `json:"deployment"`
	Certificates  []Certificate `json:"certificates,omitempty"`
	Diagnostics   []Diagnostic  `json:"diagnostics"`
	// Vulnerabilities and VulnScan are only present when an offline OSV
	// database was given (--vulndb).
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
	VulnScan        *VulnScan       `json:"vulnScan,omitempty"`
}

// Vulnerability is a known advisory affecting a library version declared or
// committed in the repository.
type Vulnerability struct {
	ID         string   `json:"id"`                // OSV id (GHSA-, PYSEC-, MAL-...)
	Aliases    []string `json:"aliases,omitempty"` // CVE and other ids
	Package    string   `json:"package"`           // Traverse coordinate, e.g. maven:g:a
	Ecosystem  string   `json:"ecosystem"`         // OSV ecosystem
	Version    string   `json:"version"`
	Severity   string   `json:"severity"` // critical | high | medium | low | unknown
	Score      float64  `json:"score,omitempty"`
	Vector     string   `json:"vector,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Fixed      []string `json:"fixed,omitempty"`
	Malicious  bool     `json:"malicious,omitempty"`
	Confidence string   `json:"confidence"` // high | medium | low
	Kind       string   `json:"kind"`       // declared | vendored-jar | assembly-ref
	Scope      string   `json:"scope,omitempty"`
	Modules    []string `json:"modules"`
	Published  string   `json:"published,omitempty"`
	File       string   `json:"file"`
	Line       int      `json:"line,omitempty"`
	LockFile   string   `json:"lockFile,omitempty"` // lockfile that pinned the evaluated version
}

// VulnScan records which database was used and what could not be evaluated.
type VulnScan struct {
	Database    string         `json:"database"`
	AsOf        string         `json:"asOf"`
	Advisories  map[string]int `json:"advisories"` // per OSV ecosystem whose full dump was loaded
	Checked     int            `json:"checked"`    // distinct package@version evaluated
	Unevaluated []Unevaluated  `json:"unevaluated"`
}

type Unevaluated struct {
	Package string `json:"package"`
	Version string `json:"version,omitempty"`
	Reason  string `json:"reason"` // missing | external-bom | unresolved | range | ecosystem-not-loaded
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
}

// Deployment is the infrastructure-as-code view: container images, Compose
// services and Kubernetes/OpenShift objects found in the repository.
type Deployment struct {
	Nodes []DeployNode `json:"nodes"`
	Links []DeployLink `json:"links"`
}

type DeployNode struct {
	ID        string   `json:"id"`
	Platform  string   `json:"platform"` // docker | compose | kubernetes | openshift | helm
	Kind      string   `json:"kind"`     // image | service | Deployment | StatefulSet | CronJob | Service | Ingress | Route | ConfigMap | Secret ...
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Images    []string `json:"images,omitempty"` // repository:tag, registry host removed
	Base      []string `json:"baseImages,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Replicas  string   `json:"replicas,omitempty"`
	Schedule  string   `json:"schedule,omitempty"`
	EnvKeys   []string `json:"envKeys,omitempty"` // names only, never values
	Paths     []string `json:"paths,omitempty"`
	Hosts     []string `json:"hosts,omitempty"` // pseudonyms only (see secure package)
	User      string   `json:"user,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Module    string   `json:"module,omitempty"` // code module built or deployed here
	File      string   `json:"file"`
	Line      int      `json:"line"`
}

type DeployLink struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // depends_on | routes | selects | env-ref | config | scales
}

type ToolInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	RulesVersion string `json:"rulesVersion"`
}

type RepoInfo struct {
	Name      string   `json:"name"`
	VCS       string   `json:"vcs"`                // git | svn | cvs | none
	Revision  string   `json:"revision,omitempty"` // git commit when readable
	Branch    string   `json:"branch,omitempty"`
	Layout    string   `json:"layout,omitempty"` // e.g. "svn-standard (trunk only)"
	ScannedAt string   `json:"scannedAt"`
	Group     string   `json:"group,omitempty"` // business domain / owner from manifest
	Notes     []string `json:"notes,omitempty"`
	// Survey records how the repository structure was read before the
	// analysis: code roots, main line, copies left out, non-code areas.
	Survey *Survey `json:"survey,omitempty"`
}

// Certificate is a public certificate found in the repository: a certificate
// file, a trusted entry of a truststore or the chain of a key entry. Only
// public data is read; subjects of personal certificates are never kept and
// host names go through the same encryption as every other host.
type Certificate struct {
	File     string `json:"file"`
	Alias    string `json:"alias,omitempty"`
	Role     string `json:"role"`              // identity (has private key) | trusted (truststore entry) | file
	Subject  string `json:"subject,omitempty"` // CN / O / OU of non-personal certificates
	Issuer   string `json:"issuer,omitempty"`
	NotAfter string `json:"notAfter,omitempty"` // YYYY-MM-DD
	CA       bool   `json:"ca,omitempty"`
	Personal bool   `json:"personal,omitempty"` // a natural person's certificate: subject omitted
	Host     string `json:"host,omitempty"`     // host-like CN, encrypted
	Status   string `json:"status,omitempty"`   // expired | expiring (within 90 days of the scan)
}

// Survey is the result of the structure reconnaissance done before scanning.
type Survey struct {
	Decision   string         `json:"decision"`             // one-line explanation of what was analyzed
	Roots      []SurveyRoot   `json:"roots"`                // code roots that were analyzed
	Copies     []SurveyCopy   `json:"copies,omitempty"`     // copies of a main line left out
	NonCode    []SurveyArea   `json:"nonCode,omitempty"`    // documentation, models, binaries, other
	BinaryOnly []SurveyBinary `json:"binaryOnly,omitempty"` // modules that ship only built artifacts
	AllCopies  bool           `json:"allCopies,omitempty"`  // --layout all: copies were analyzed too
}

type SurveyRoot struct {
	Path    string `json:"path"` // relative, "" = repository root
	Coord   string `json:"coord,omitempty"`
	Version string `json:"version,omitempty"`
	Files   int    `json:"files"`
	Main    bool   `json:"main,omitempty"`   // chosen main line of a group of copies
	Reason  string `json:"reason,omitempty"` // why it was chosen as main line
}

type SurveyCopy struct {
	Path      string  `json:"path"`
	Of        string  `json:"of"` // main line it copies
	Version   string  `json:"version,omitempty"`
	Files     int     `json:"files"`
	Identical bool    `json:"identical,omitempty"`
	Overlap   float64 `json:"overlap"` // Jaccard similarity of source paths with the main line
}

type SurveyArea struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"` // docs | models | binaries | data | other
	Files int    `json:"files"`
}

type SurveyBinary struct {
	Path      string   `json:"path"`
	Artifacts []string `json:"artifacts"`
}

type LangStat struct {
	Language string `json:"language"`
	Category string `json:"category"`
	Files    int    `json:"files"`
	Code     int    `json:"code"`
	Comment  int    `json:"comment"`
	Blank    int    `json:"blank"`
}

type File struct {
	Path      string `json:"path"`
	Language  string `json:"language"`
	Category  string `json:"category"`
	Bytes     int64  `json:"bytes"`
	Code      int    `json:"code"`
	Comment   int    `json:"comment"`
	Blank     int    `json:"blank"`
	SourceSet string `json:"sourceSet"` // main | test | generated
	Module    string `json:"module,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
}

// Module is a build unit (Maven module, Gradle project, npm package, .NET project...).
type Module struct {
	ID         string       `json:"id"` // coordinate used for cross-repo linking, e.g. maven:g:a
	Build      string       `json:"build"`
	Name       string       `json:"name"`
	Version    string       `json:"version,omitempty"`
	Packaging  string       `json:"packaging,omitempty"`
	Parent     string       `json:"parent,omitempty"`
	Path       string       `json:"path"` // directory relative to repo root
	Descriptor string       `json:"descriptor"`
	Runtime    string       `json:"runtime,omitempty"`  // e.g. java 1.6, net48, node
	Artifact   string       `json:"artifact,omitempty"` // built file name when known (e.g. app.war)
	Children   []string     `json:"children,omitempty"`
	Deps       []Dependency `json:"deps"`
	// Transitive lists the packages pinned by a lockfile (package-lock.json,
	// yarn.lock, pnpm-lock.yaml, bun.lock, composer.lock, packages.lock.json,
	// poetry.lock, Pipfile.lock, uv.lock, gradle.lockfile) that are not direct
	// dependencies. Used for the SBOM and vulnerability matching only.
	Transitive []Dependency `json:"transitive,omitempty"`
}

type Dependency struct {
	ID       string `json:"id"` // coordinate, same scheme as Module.ID
	Version  string `json:"version,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Managed  bool   `json:"managed,omitempty"`  // declared in dependencyManagement only
	Kind     string `json:"kind"`               // declared | vendored-jar | project-ref
	Resolved bool   `json:"resolved"`           // version fully resolved (no ${...})
	Locked   string `json:"locked,omitempty"`   // exact version pinned by a lockfile or central package management
	LockFile string `json:"lockFile,omitempty"` // file that pinned it
	Line     int    `json:"line,omitempty"`
	File     string `json:"file,omitempty"`
}

type Deployable struct {
	Module      string   `json:"module"`
	Kind        string   `json:"kind"` // war | ear | ejb | jar | exe | library | service
	ContextRoot string   `json:"contextRoot,omitempty"`
	RootSource  string   `json:"contextRootSource,omitempty"` // descriptor that set it, or "inferred"
	AppPaths    []string `json:"appPaths,omitempty"`
	Includes    []string `json:"includes,omitempty"`
	Descriptor  string   `json:"descriptor,omitempty"`
}

type Component struct {
	FQN         string   `json:"fqn"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"` // class | interface | enum | record | annotation | nested
	Language    string   `json:"language"`
	Package     string   `json:"package"`
	SourceSet   string   `json:"sourceSet"`
	Module      string   `json:"module,omitempty"`
	Layer       string   `json:"layer"`
	LayerReason string   `json:"layerReason"`
	Stereotypes []string `json:"stereotypes,omitempty"`
	Extends     []string `json:"extends,omitempty"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
}

type CodeDep struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"` // import | wildcard | same-package | fqn | extends | inject
	Confidence string `json:"confidence"`
	File       string `json:"file"`
	Line       int    `json:"line"`
}

// Contract is something a repo exposes or needs from the outside world.
// Matching contracts across repos is how the portfolio landscape is built.
type Contract struct {
	Kind       string `json:"kind"`      // http | soap | jms | jndi | db-table | ejb
	Direction  string `json:"direction"` // exposes | calls | publishes | subscribes | uses | defines
	Key        string `json:"key"`
	Match      string `json:"match"` // normalized key used for matching
	Component  string `json:"component,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Confidence string `json:"confidence"`
	// Host is the pseudonym of the remote host (never the clear-text name);
	// empty when no encryption key was provided.
	Host string `json:"host,omitempty"`
	// Transport and Auth are populated where statically detectable.
	Transport string `json:"transport,omitempty"` // plain | tls | mtls
	Auth      string `json:"auth,omitempty"`      // basic | bearer | ws-security | client-cert | ...
	// Role tells inbound interfaces meant for other systems (api) from user
	// screens (ui). Counterpart names the other side when the code does.
	Role        string `json:"role,omitempty"`
	Counterpart string `json:"counterpart,omitempty"`
	File        string `json:"file"`
	Line        int    `json:"line"`
}

type Finding struct {
	ID         string `json:"id"` // stable: hash(ruleId|repo|file), independent of line numbers
	RuleID     string `json:"ruleId"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Technology string `json:"technology,omitempty"`
	Severity   string `json:"severity"` // info | low | medium | high
	Confidence string `json:"confidence"`
	Message    string `json:"message,omitempty"`
	File       string `json:"file"`
	Lines      []int  `json:"lines,omitempty"`
	Count      int    `json:"count"`
	SourceSet  string `json:"sourceSet"`
	// AuthKind classifies authentication-related findings: user (login),
	// service (system to system), signature (e-signature) or transport (TLS).
	AuthKind string `json:"authKind,omitempty"`
}

type PackageStat struct {
	Package     string  `json:"package"`
	Module      string  `json:"module,omitempty"`
	Layer       string  `json:"layer"`
	Types       int     `json:"types"`
	Afferent    int     `json:"ca"`
	Efferent    int     `json:"ce"`
	Instability float64 `json:"instability"`
}

type Cycle struct {
	Level   string   `json:"level"` // package | module
	Members []string `json:"members"`
}

type LayerIssue struct {
	FromLayer string `json:"fromLayer"`
	ToLayer   string `json:"toLayer"`
	From      string `json:"from"`
	To        string `json:"to"`
	File      string `json:"file"`
	Line      int    `json:"line"`
}

type Diagnostic struct {
	File    string `json:"file,omitempty"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}
