// Package vuln matches declared library versions against an offline OSV
// (https://ossf.github.io/osv-schema/) database. It never uses the network.
package vuln

import (
	"regexp"
	"strings"
)

// Compare orders two versions with the rules of an OSV ecosystem.
// It returns -1, 0 or 1.
func Compare(eco, a, b string) int {
	switch eco {
	case "npm", "Go", "crates.io":
		if x, ok := parseSemver(a, false); ok {
			if y, ok := parseSemver(b, false); ok {
				return x.cmp(y)
			}
		}
	case "NuGet":
		if x, ok := parseSemver(a, true); ok {
			if y, ok := parseSemver(b, true); ok {
				return x.cmp(y)
			}
		}
	case "PyPI":
		if x, ok := parsePEP440(a); ok {
			if y, ok := parsePEP440(b); ok {
				return x.cmp(y)
			}
		}
	}
	return compareMaven(a, b)
}

// ---------- Maven ComparableVersion (also the generic fallback) ----------

type mKind int

const (
	mInt mKind = iota
	mStr
	mList
)

type mItem struct {
	kind mKind
	num  string // digits without leading zeros ("" means 0)
	str  string
	list []*mItem
}

var mQualifiers = []string{"alpha", "beta", "milestone", "rc", "snapshot", "", "sp"}

func mComparable(q string) string {
	for i, x := range mQualifiers {
		if x == q {
			return string(rune('0' + i))
		}
	}
	return "7-" + q
}

func newMStr(s string, followedByDigit bool) *mItem {
	if followedByDigit && len(s) == 1 {
		switch s {
		case "a":
			s = "alpha"
		case "b":
			s = "beta"
		case "m":
			s = "milestone"
		}
	}
	switch s {
	case "ga", "final", "release":
		s = ""
	case "cr":
		s = "rc"
	}
	return &mItem{kind: mStr, str: s}
}

func newMInt(s string) *mItem {
	return &mItem{kind: mInt, num: strings.TrimLeft(s, "0")}
}

func (it *mItem) isNull() bool {
	switch it.kind {
	case mInt:
		return it.num == ""
	case mStr:
		return it.str == ""
	}
	return len(it.list) == 0
}

func (it *mItem) normalize() {
	for i := len(it.list) - 1; i >= 0; i-- {
		x := it.list[i]
		if x.isNull() {
			it.list = append(it.list[:i], it.list[i+1:]...)
		} else if x.kind != mList {
			break
		}
	}
}

func parseMaven(v string) *mItem {
	v = strings.ToLower(strings.TrimSpace(v))
	root := &mItem{kind: mList}
	list := root
	stack := []*mItem{root}
	isDigit := false
	start := 0
	parseItem := func(digit bool, s string) *mItem {
		if digit {
			return newMInt(s)
		}
		return newMStr(s, false)
	}
	push := func() {
		n := &mItem{kind: mList}
		list.list = append(list.list, n)
		list = n
		stack = append(stack, n)
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c == '.':
			if i == start {
				list.list = append(list.list, newMInt("0"))
			} else {
				list.list = append(list.list, parseItem(isDigit, v[start:i]))
			}
			start = i + 1
		case c == '-':
			if i == start {
				list.list = append(list.list, newMInt("0"))
			} else {
				list.list = append(list.list, parseItem(isDigit, v[start:i]))
			}
			start = i + 1
			push()
		case c >= '0' && c <= '9':
			if !isDigit && i > start {
				list.list = append(list.list, newMStr(v[start:i], true))
				start = i
				push()
			}
			isDigit = true
		default:
			if isDigit && i > start {
				list.list = append(list.list, parseItem(true, v[start:i]))
				start = i
				push()
			}
			isDigit = false
		}
	}
	if len(v) > start {
		list.list = append(list.list, parseItem(isDigit, v[start:]))
	}
	for i := len(stack) - 1; i >= 0; i-- {
		stack[i].normalize()
	}
	return root
}

func cmpNum(a, b string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// cmpItem compares a with b; b may be nil (padding).
func cmpItem(a, b *mItem) int {
	switch a.kind {
	case mInt:
		if b == nil {
			if a.num == "" {
				return 0
			}
			return 1
		}
		if b.kind == mInt {
			return cmpNum(a.num, b.num)
		}
		return 1
	case mStr:
		if b == nil {
			return strings.Compare(mComparable(a.str), mComparable(""))
		}
		switch b.kind {
		case mInt, mList:
			return -1
		}
		return strings.Compare(mComparable(a.str), mComparable(b.str))
	}
	// list
	if b == nil {
		if len(a.list) == 0 {
			return 0
		}
		return cmpItem(a.list[0], nil)
	}
	switch b.kind {
	case mInt:
		return -1
	case mStr:
		return 1
	}
	for i := 0; i < len(a.list) || i < len(b.list); i++ {
		var l, r *mItem
		if i < len(a.list) {
			l = a.list[i]
		}
		if i < len(b.list) {
			r = b.list[i]
		}
		var c int
		if l == nil {
			if r != nil {
				c = -cmpItem(r, nil)
			}
		} else {
			c = cmpItem(l, r)
		}
		if c != 0 {
			return c
		}
	}
	return 0
}

func compareMaven(a, b string) int {
	c := cmpItem(parseMaven(a), parseMaven(b))
	switch {
	case c < 0:
		return -1
	case c > 0:
		return 1
	}
	return 0
}

// ---------- SemVer 2.0 (npm, Go, crates.io) and NuGet ----------

type semver struct {
	nums []string
	pre  []string
}

var reSemver = regexp.MustCompile(`^v?(\d+(?:\.\d+)*)(?:-([0-9A-Za-z.\-]+))?(?:\+[0-9A-Za-z.\-]+)?$`)

func parseSemver(s string, fold bool) (semver, bool) {
	m := reSemver.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{}, false
	}
	v := semver{}
	for _, n := range strings.Split(m[1], ".") {
		v.nums = append(v.nums, strings.TrimLeft(n, "0"))
	}
	if m[2] != "" {
		p := m[2]
		if fold {
			p = strings.ToLower(p)
		}
		v.pre = strings.Split(p, ".")
	}
	return v, true
}

func isNum(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (a semver) cmp(b semver) int {
	for i := 0; i < len(a.nums) || i < len(b.nums); i++ {
		x, y := "", ""
		if i < len(a.nums) {
			x = a.nums[i]
		}
		if i < len(b.nums) {
			y = b.nums[i]
		}
		if c := cmpNum(x, y); c != 0 {
			return c
		}
	}
	switch {
	case len(a.pre) == 0 && len(b.pre) == 0:
		return 0
	case len(a.pre) == 0:
		return 1
	case len(b.pre) == 0:
		return -1
	}
	for i := 0; i < len(a.pre) && i < len(b.pre); i++ {
		x, y := a.pre[i], b.pre[i]
		nx, ny := isNum(x), isNum(y)
		var c int
		switch {
		case nx && ny:
			c = cmpNum(strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0"))
		case nx:
			c = -1
		case ny:
			c = 1
		default:
			c = strings.Compare(x, y)
		}
		if c != 0 {
			return c
		}
	}
	switch {
	case len(a.pre) < len(b.pre):
		return -1
	case len(a.pre) > len(b.pre):
		return 1
	}
	return 0
}

// ---------- PEP 440 (PyPI) ----------

var rePEP440 = regexp.MustCompile(`(?i)^v?(?:(\d+)!)?(\d+(?:\.\d+)*)` +
	`(?:[-_.]?(a|alpha|b|beta|c|rc|pre|preview)[-_.]?(\d*))?` +
	`(?:-(\d+)|[-_.]?(post|rev|r)[-_.]?(\d*))?` +
	`(?:[-_.]?(dev)[-_.]?(\d*))?` +
	`(?:\+[a-z0-9]+(?:[-_.][a-z0-9]+)*)?$`)

const (
	negInf = -1 << 62
	posInf = 1 << 62
)

type pep440 struct {
	epoch   int
	release []string
	pre     [2]int // phase, number
	post    int
	dev     int
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s) && i < 18; i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func parsePEP440(s string) (pep440, bool) {
	m := rePEP440.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return pep440{}, false
	}
	v := pep440{epoch: atoi(m[1])}
	for _, n := range strings.Split(m[2], ".") {
		v.release = append(v.release, strings.TrimLeft(n, "0"))
	}
	for len(v.release) > 1 && v.release[len(v.release)-1] == "" {
		v.release = v.release[:len(v.release)-1]
	}
	hasPre, hasPost, hasDev := m[3] != "", m[5] != "" || m[6] != "", m[8] != ""
	if hasPre {
		ph := 0
		switch strings.ToLower(m[3]) {
		case "b", "beta":
			ph = 1
		case "c", "rc", "pre", "preview":
			ph = 2
		}
		v.pre = [2]int{ph, atoi(m[4])}
	} else if !hasPost && hasDev {
		v.pre = [2]int{negInf, 0}
	} else {
		v.pre = [2]int{posInf, 0}
	}
	switch {
	case m[5] != "":
		v.post = atoi(m[5])
	case m[6] != "":
		v.post = atoi(m[7])
	default:
		v.post = negInf
	}
	if hasDev {
		v.dev = atoi(m[9])
	} else {
		v.dev = posInf
	}
	return v, true
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func (a pep440) cmp(b pep440) int {
	if c := cmpInt(a.epoch, b.epoch); c != 0 {
		return c
	}
	for i := 0; i < len(a.release) || i < len(b.release); i++ {
		x, y := "", ""
		if i < len(a.release) {
			x = a.release[i]
		}
		if i < len(b.release) {
			y = b.release[i]
		}
		if c := cmpNum(x, y); c != 0 {
			return c
		}
	}
	if c := cmpInt(a.pre[0], b.pre[0]); c != 0 {
		return c
	}
	if c := cmpInt(a.pre[1], b.pre[1]); c != 0 {
		return c
	}
	if c := cmpInt(a.post, b.post); c != 0 {
		return c
	}
	return cmpInt(a.dev, b.dev)
}

// ---------- exact versions ----------

var reExact = regexp.MustCompile(`^\d[0-9A-Za-z.\-_+]*$`)

// Exact returns the concrete version written in a descriptor, or a reason why
// it cannot be evaluated: missing | unresolved | range.
func Exact(v string) (string, string) {
	v = strings.TrimSpace(v)
	switch {
	case v == "":
		return "", "missing"
	case strings.Contains(v, "${") || strings.Contains(v, "$("):
		return "", "unresolved"
	}
	if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") && !strings.Contains(v, ",") {
		v = strings.TrimSpace(v[1 : len(v)-1]) // NuGet / Maven exact pin: [1.2.3]
	}
	v = strings.TrimPrefix(strings.TrimPrefix(v, "=="), "=")
	v = strings.TrimSpace(v)
	if len(v) > 1 && (v[0] == 'v' || v[0] == 'V') && v[1] >= '0' && v[1] <= '9' {
		v = v[1:]
	}
	if strings.HasSuffix(v, ".+") || strings.HasSuffix(v, ".x") || strings.HasSuffix(v, ".*") || strings.ContainsAny(v, "^~<>*|, ") {
		return "", "range"
	}
	if !reExact.MatchString(v) {
		return "", "range"
	}
	return v, ""
}
