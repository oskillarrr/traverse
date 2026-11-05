// Package rules implements a declarative, data-driven rule engine.
// Rules are regular expressions (RE2 syntax) applied to comment-free text,
// or to file paths. Every rule carries examples that are verified by
// `traverse rules test`, so contributions are self-documenting and testable.
package rules

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/lang"
)

//go:embed default.json
var defaultCatalog []byte

type Examples struct {
	Match   []string `json:"match,omitempty"`
	NoMatch []string `json:"noMatch,omitempty"`
}

type Rule struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Technology  string   `json:"technology"`
	Severity    string   `json:"severity"`
	Confidence  string   `json:"confidence"`
	Languages   []string `json:"languages,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	PathPattern string   `json:"pathPattern,omitempty"`
	// Exclude drops a match whose matched text also matches this regex
	// (for example a URL where a password was expected).
	Exclude string   `json:"exclude,omitempty"`
	Message string   `json:"message,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	// AuthKind classifies authentication findings: user (login), service
	// (service to service), signature (e-signature) or transport (TLS/mTLS).
	// Empty: derived from the category.
	AuthKind string   `json:"authKind,omitempty"`
	Examples Examples `json:"examples,omitempty"`

	re, pathRe, exRe *regexp.Regexp
	langs            map[string]bool
	need             []string // prefilter literals (nil = always run)
	fold             bool
}

type Catalog struct {
	Version string  `json:"version"`
	Rules   []*Rule `json:"rules"`
}

// Load returns the embedded catalog (unless disabled) plus extra rule files.
// Rules in later files override earlier ones with the same id.
func Load(useDefault bool, extra []string) (*Catalog, error) {
	c := &Catalog{}
	byID := map[string]int{}
	merge := func(data []byte, src string) error {
		var in Catalog
		if err := json.Unmarshal(data, &in); err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		if c.Version == "" {
			c.Version = in.Version
		} else if in.Version != "" {
			c.Version += "+" + in.Version
		}
		for _, r := range in.Rules {
			if err := r.compile(); err != nil {
				return fmt.Errorf("%s: rule %s: %w", src, r.ID, err)
			}
			if i, ok := byID[r.ID]; ok {
				c.Rules[i] = r
				continue
			}
			byID[r.ID] = len(c.Rules)
			c.Rules = append(c.Rules, r)
		}
		return nil
	}
	if useDefault {
		if err := merge(defaultCatalog, "embedded default.json"); err != nil {
			return nil, err
		}
	}
	for _, f := range extra {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		if err := merge(b, f); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(c.Rules, func(i, j int) bool { return c.Rules[i].ID < c.Rules[j].ID })
	return c, nil
}

func (r *Rule) compile() error {
	if r.ID == "" || (r.Pattern == "" && r.PathPattern == "") {
		return fmt.Errorf("id and pattern or pathPattern are required")
	}
	var err error
	if r.Pattern != "" {
		if r.re, err = regexp.Compile(r.Pattern); err != nil {
			return err
		}
		r.need, r.fold = requiredLiterals(r.Pattern)
	}
	if r.PathPattern != "" {
		if r.pathRe, err = regexp.Compile(r.PathPattern); err != nil {
			return err
		}
	}
	if r.Exclude != "" {
		if r.Pattern == "" {
			return fmt.Errorf("exclude needs a pattern")
		}
		if r.exRe, err = regexp.Compile(r.Exclude); err != nil {
			return err
		}
	}
	switch r.AuthKind {
	case "", "user", "service", "signature", "transport":
	default:
		return fmt.Errorf("authKind %q must be user, service, signature or transport", r.AuthKind)
	}
	if r.Severity == "" {
		r.Severity = "info"
	}
	if r.Confidence == "" {
		r.Confidence = "medium"
	}
	r.langs = map[string]bool{}
	for _, l := range r.Languages {
		r.langs[l] = true
	}
	return nil
}

// Hit is a rule match in one file.
type Hit struct {
	Rule  *Rule
	Lines []int
	Count int
}

const maxLines = 50

// Apply evaluates all rules against one file. code must be comment-free text
// (string literals kept) with the same offsets as the source.
func (c *Catalog) Apply(rel, language string, s *lang.Stripped) []Hit {
	var hits []Hit
	lower := ""
	if s != nil {
		lower = strings.ToLower(s.Code)
	}
	for _, r := range c.Rules {
		if r.pathRe != nil && r.re == nil {
			if r.pathRe.MatchString(rel) {
				hits = append(hits, Hit{Rule: r, Count: 1})
			}
			continue
		}
		if s == nil || r.re == nil {
			continue
		}
		if len(r.langs) > 0 && !r.langs[language] {
			continue
		}
		if r.pathRe != nil && !r.pathRe.MatchString(rel) {
			continue
		}
		if !r.mayMatch(s.Code, lower) {
			continue
		}
		locs := r.matches(s.Code)
		if len(locs) == 0 {
			continue
		}
		h := Hit{Rule: r, Count: len(locs)}
		last := -1
		for _, l := range locs {
			ln := s.Line(l[0])
			if ln != last && len(h.Lines) < maxLines {
				h.Lines = append(h.Lines, ln)
			}
			last = ln
		}
		hits = append(hits, h)
	}
	return hits
}

// matches returns the match positions that the rule's exclude pattern does
// not discard.
func (r *Rule) matches(text string) [][]int {
	locs := r.re.FindAllStringIndex(text, -1)
	if r.exRe == nil {
		return locs
	}
	kept := locs[:0]
	for _, l := range locs {
		if !r.exRe.MatchString(text[l[0]:l[1]]) {
			kept = append(kept, l)
		}
	}
	return kept
}

// SelfTest verifies every rule example. Examples are lexed with the syntax of
// the rule's first language, so "noMatch" can prove comments are ignored.
func (c *Catalog) SelfTest() []error {
	var errs []error
	for _, r := range c.Rules {
		if r.re == nil {
			continue
		}
		syn := lang.None
		if len(r.Languages) > 0 {
			if l, ok := lang.ByName(r.Languages[0]); ok {
				syn = l.Syntax
			}
		}
		for _, ex := range r.Examples.Match {
			if len(r.matches(lang.Strip(ex, syn).Code)) == 0 {
				errs = append(errs, fmt.Errorf("%s: expected match: %q", r.ID, ex))
			}
		}
		for _, ex := range r.Examples.NoMatch {
			if len(r.matches(lang.Strip(ex, syn).Code)) > 0 {
				errs = append(errs, fmt.Errorf("%s: unexpected match: %q", r.ID, ex))
			}
		}
	}
	return errs
}
