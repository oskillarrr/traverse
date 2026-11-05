package model

// Portfolio is the cross-repository view built from several Facts.
type Portfolio struct {
	SchemaVersion   string         `json:"schemaVersion"`
	Tool            ToolInfo       `json:"tool"`
	GeneratedAt     string         `json:"generatedAt"`
	Systems         []System       `json:"systems"`
	Links           []Link         `json:"links"`
	SharedTables    []SharedTable  `json:"sharedTables"`
	LibraryVersions []LibraryUsage `json:"libraryVersions"`
	External        []ExternalNeed `json:"external"`
	Technologies    []TechUsage    `json:"technologies"`
	Vulnerabilities []VulnUsage    `json:"vulnerabilities,omitempty"`
	VulnDB          string         `json:"vulnDatabase,omitempty"`
	VulnAsOf        string         `json:"vulnAsOf,omitempty"`
}

// VulnUsage is one advisory and the systems (and versions) it affects.
type VulnUsage struct {
	ID        string   `json:"id"`
	Aliases   []string `json:"aliases,omitempty"`
	Package   string   `json:"package"`
	Severity  string   `json:"severity"`
	Score     float64  `json:"score,omitempty"`
	Malicious bool     `json:"malicious,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Fixed     []string `json:"fixed,omitempty"`
	Affected  []string `json:"affected"` // system@version
}

type System struct {
	Name        string         `json:"name"`
	Group       string         `json:"group,omitempty"`
	VCS         string         `json:"vcs"`
	Revision    string         `json:"revision,omitempty"`
	Languages   []string       `json:"languages"`
	Code        int            `json:"code"`
	Files       int            `json:"files"`
	Modules     int            `json:"modules"`
	Components  int            `json:"components"`
	Exposes     int            `json:"exposes"`
	Calls       int            `json:"calls"`
	Tables      int            `json:"tables"`
	Runtimes    []string       `json:"runtimes,omitempty"`
	Deployables []string       `json:"deployables,omitempty"`
	Risks       map[string]int `json:"risks"`
	Cycles      int            `json:"cycles"`
	LayerIssues int            `json:"layerIssues"`
	Vulns       map[string]int `json:"vulnerabilities,omitempty"` // severity -> count
}

type Link struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	Kind       string   `json:"kind"` // library | http | soap | jms | kafka | ejb | shared-db | shared-datasource
	Keys       []string `json:"keys"`
	Count      int      `json:"count"`
	Confidence string   `json:"confidence"`
	// Candidates > 1: the interface the consumer calls is implemented by
	// that many analyzed systems (a callback contract, a shared WSDL); which
	// one is called is decided by run-time configuration.
	Candidates int `json:"candidates,omitempty"`
}

type SharedTable struct {
	Table   string   `json:"table"`
	Basis   string   `json:"basis"` // schema | datasource | name-only
	Systems []string `json:"systems"`
	Access  []string `json:"access"` // system:read|write|defines|uses
}

type LibraryUsage struct {
	Library  string              `json:"library"`
	Internal bool                `json:"internal"` // produced by a system of the portfolio
	Versions map[string][]string `json:"versions"` // version -> systems
}

type ExternalNeed struct {
	Kind    string   `json:"kind"`
	Key     string   `json:"key"`
	Host    string   `json:"host,omitempty"` // pseudonym
	Systems []string `json:"systems"`
}

type TechUsage struct {
	Technology string         `json:"technology"`
	Category   string         `json:"category"`
	Severity   string         `json:"severity"`
	Systems    map[string]int `json:"systems"` // system -> files
}
