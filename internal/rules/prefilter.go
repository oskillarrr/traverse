package rules

import (
	"regexp/syntax"
	"strings"
)

// requiredLiterals derives, from a regular expression, a set of literals such
// that every match must contain at least one of them. Scanning a file with
// strings.Contains is much cheaper than running the regex, so files that
// contain none of the literals are skipped. Returns nil when no safe set exists.
// fold reports that the literals must be compared case-insensitively.
func requiredLiterals(pattern string) (lits []string, fold bool) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, false
	}
	set, f := req(re.Simplify())
	for _, l := range set {
		if len(l) < 2 { // too short to be selective
			return nil, false
		}
	}
	return set, f
}

func req(re *syntax.Regexp) ([]string, bool) {
	switch re.Op {
	case syntax.OpLiteral:
		s := string(re.Rune)
		if re.Flags&syntax.FoldCase != 0 {
			return []string{strings.ToLower(s)}, true
		}
		return []string{s}, false
	case syntax.OpCapture:
		return req(re.Sub[0])
	case syntax.OpPlus:
		return req(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min >= 1 {
			return req(re.Sub[0])
		}
	case syntax.OpConcat:
		var best []string
		bestFold, bestScore := false, 0
		for _, s := range re.Sub {
			l, f := req(s)
			if l == nil {
				continue
			}
			score := minLen(l)
			if score > bestScore {
				best, bestFold, bestScore = l, f, score
			}
		}
		return best, bestFold
	case syntax.OpAlternate:
		var all []string
		fold := false
		for _, s := range re.Sub {
			l, f := req(s)
			if l == nil {
				return nil, false
			}
			all = append(all, l...)
			fold = fold || f
		}
		if fold { // mixing: compare everything case-insensitively
			for i := range all {
				all[i] = strings.ToLower(all[i])
			}
		}
		return all, fold
	}
	return nil, false
}

func minLen(xs []string) int {
	m := 1 << 30
	for _, x := range xs {
		if len(x) < m {
			m = len(x)
		}
	}
	return m
}

func (r *Rule) mayMatch(text, lower string) bool {
	if r.need == nil {
		return true
	}
	hay := text
	if r.fold {
		hay = lower
	}
	for _, l := range r.need {
		if strings.Contains(hay, l) {
			return true
		}
	}
	return false
}
