// Package sqlparse extracts table references from text that looks like SQL.
// It is a shared, read-only helper used by the source scanner (SQL in string
// literals, scripts and embedded SQL) and by the descriptor parser (iBATIS /
// MyBatis mapper statements). It never connects anywhere and never executes
// SQL.
package sqlparse

import (
	"regexp"
	"strings"
)

// Ref is one table reference with the access it implies.
type Ref struct {
	Table string // as written (may be schema-qualified)
	Op    string // read | write
}

var (
	reStmt  = regexp.MustCompile(`(?is)\b(select\b.+\bfrom|insert\s+into|delete\s+from|update\s+[A-Za-z_][\w$#.]*\s+set|merge\s+into)\b`)
	reStart = regexp.MustCompile(`(?i)^\s*(from|join|inner\s+join|left\s+(outer\s+)?join|where|and|or|set|values)\b`)
	reTable = regexp.MustCompile(`(?i)\b(from|join|into|update)\s+([A-Za-z_][\w$#]*(?:\.[A-Za-z_][\w$#]*)?)`)
	reWrite = regexp.MustCompile(`(?i)^\s*(insert|update|delete|merge)\b`)
	reAlias = regexp.MustCompile(`(?i)\b(from|join|update|into)\s+([A-Za-z_][\w$#.]*)\s+(?:as\s+)?([A-Za-z_]\w*)`)
	reFrom  = regexp.MustCompile(`(?is)\bfrom\s+(.+?)(?:\bwhere\b|\bgroup\s+by\b|\border\s+by\b|\bunion\b|\bhaving\b|\bconnect\s+by\b|\bstart\s+with\b|$)`)
)

// keywords that can follow a table name and are not aliases
var notAlias = map[string]bool{"WHERE": true, "ON": true, "JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true, "FULL": true,
	"OUTER": true, "CROSS": true, "NATURAL": true, "GROUP": true, "ORDER": true, "HAVING": true, "UNION": true, "SET": true,
	"VALUES": true, "START": true, "CONNECT": true, "USING": true, "FOR": true, "LIMIT": true, "FETCH": true, "OFFSET": true,
	"PARTITION": true, "WITH": true, "MINUS": true, "INTERSECT": true, "EXCEPT": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "AS": true, "AND": true, "OR": true, "NOT": true, "SELECT": true}

// aliases returns the correlation names declared in a statement (FROM T t,
// JOIN u.roles r…): a qualified reference whose qualifier is an alias is a
// column or a JPQL/HQL navigation path, not schema.table.
func aliases(s string) map[string]bool {
	out := map[string]bool{}
	for _, m := range reAlias.FindAllStringSubmatch(s, -1) {
		if a := strings.ToUpper(m[3]); !notAlias[a] {
			out[a] = true
		}
	}
	for _, fm := range reFrom.FindAllStringSubmatch(s, -1) {
		for _, item := range topLevelSplit(fm[1]) {
			f := strings.Fields(strings.TrimSpace(item))
			if len(f) >= 2 && !strings.HasPrefix(f[0], "(") {
				a := strings.ToUpper(f[1])
				if a == "AS" && len(f) >= 3 {
					a = strings.ToUpper(f[2])
				}
				if !notAlias[a] {
					out[a] = true
				}
			}
		}
	}
	return out
}

// commaTables returns the 2nd..nth tables of comma-separated FROM lists.
func commaTables(s string) []string {
	var out []string
	for _, fm := range reFrom.FindAllStringSubmatch(s, -1) {
		items := topLevelSplit(fm[1])
		for _, item := range items[min(1, len(items)):] {
			f := strings.Fields(strings.TrimSpace(item))
			if len(f) > 0 && !strings.HasPrefix(f[0], "(") && reIdent.MatchString(f[0]) {
				out = append(out, f[0])
			}
		}
	}
	return out
}

var reIdent = regexp.MustCompile(`^[A-Za-z_][\w$#]*(?:\.[A-Za-z_][\w$#]*)?$`)

func topLevelSplit(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 { // end of an enclosing subquery
				return append(out, s[start:i])
			}
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

// noise lists keywords and pseudo-tables the FROM/JOIN heuristic can latch onto.
var noise = map[string]bool{
	"SELECT": true, "DUAL": true, "WHERE": true, "SET": true, "VALUES": true, "TABLE": true, "THE": true, "LATERAL": true,
	"UNNEST": true, "JSON_TABLE": true, "SYSIBM.SYSDUMMY1": true, "SYSDUMMY1": true, "A": true, "AN": true, "THIS": true,
	"FETCH": true, "FIRST": true, "NEXT": true, "ONLY": true, "ROWS": true, "OFFSET": true, "LIMIT": true, "DISTINCT": true,
	"OBJECT": true, "NEW": true,
}

// LooksLikeSQL reports whether s is worth scanning for tables.
func LooksLikeSQL(s string) bool {
	return len(s) >= 8 && (reStmt.MatchString(s) || reStart.MatchString(s))
}

// Tables extracts the distinct table references from a SQL-like string. The
// statement's leading verb sets the default access; INTO and UPDATE targets are
// always writes.
func Tables(s string) []Ref {
	if !LooksLikeSQL(s) {
		return nil
	}
	op := "read"
	if reWrite.MatchString(s) {
		op = "write"
	}
	var out []Ref
	seen := map[string]bool{}
	al := aliases(s)
	add := func(t, o string) {
		u := strings.ToUpper(t)
		if noise[u] || seen[u] {
			return
		}
		if i := strings.IndexByte(u, '.'); i > 0 && al[u[:i]] {
			return // alias.column or JPQL navigation (join u.roles r)
		}
		seen[u] = true
		out = append(out, Ref{t, o})
	}
	for _, m := range reTable.FindAllStringSubmatch(s, -1) {
		o := op
		if k := strings.ToLower(m[1]); k == "into" || k == "update" {
			o = "write"
		}
		add(m[2], o)
	}
	for _, t := range commaTables(s) {
		add(t, "read")
	}
	return out
}

// Normalize upper-cases a table name, strips quoting and keeps the base name
// after the last schema separator.
func Normalize(t string) (full, base string) {
	full = strings.ToUpper(strings.Trim(t, `"[]`))
	base = full
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		base = base[i+1:]
	}
	return full, base
}
