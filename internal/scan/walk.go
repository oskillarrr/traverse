package scan

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// DefaultExcludedDirs are skipped at any depth: VCS metadata, IDE files,
// build outputs and third-party dependency folders.
var DefaultExcludedDirs = []string{
	".git", ".svn", ".hg", ".bzr", "CVS", "_darcs",
	".idea", ".vscode", ".settings", ".metadata", ".vs",
	".gradle", ".mvn", ".m2", "node_modules", "bower_components", "jspm_packages",
	"target", "build", "dist", "out", "bin", "obj", "classes", "coverage",
	".next", ".nuxt", ".angular", ".cache", "__pycache__", ".venv", "venv", ".tox",
	"vendor", "packages",
}

type walkOptions struct {
	excludeDirs  map[string]bool
	excludeGlobs []string
	svnAll       bool
	skip         []string // relative directories left out (copies found by the survey)
}

// genericNames are folder names that say where the code is, not what it is
// (in several languages).
var genericNames = map[string]bool{"fuente": true, "fuentes": true, "src": true, "source": true, "sources": true,
	"codigo": true, "código": true, "code": true, "trunk": true, "repo": true, "repositorio": true, "fonte": true,
	"fontes": true, "quellcode": true, "sorgenti": true, "sourcecode": true, "source-code": true, "checkout": true}

// SystemName names a system after its folder, skipping generic folder names
// such as "fuente" or "trunk" (…/orders/fuente → "orders").
func SystemName(abs string) string {
	n := filepath.Base(abs)
	for genericNames[strings.ToLower(n)] {
		p := filepath.Dir(abs)
		if p == abs || filepath.Base(p) == "" || filepath.Base(p) == string(filepath.Separator) {
			break
		}
		abs, n = p, filepath.Base(p)
	}
	return n
}

type walkedFile struct {
	abs, rel string
	size     int64
}

type walkResult struct {
	files   []walkedFile
	skipped []string // svn branches/tags directories skipped
}

// walk enumerates files, pruning excluded directories during traversal and
// skipping Subversion branches/tags copies unless svnAll is set.
func walk(root string, o walkOptions) (walkResult, error) {
	var res walkResult
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entry: ignore, reported by caller if needed
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p == root {
				return nil
			}
			name := d.Name()
			if o.excludeDirs[name] || matchGlobs(rel+"/", o.excludeGlobs) {
				return filepath.SkipDir
			}
			for _, s := range o.skip {
				if rel == s {
					return filepath.SkipDir
				}
			}
			if !o.svnAll && (name == "branches" || name == "tags") && isSVNStandardParent(filepath.Dir(p)) {
				res.skipped = append(res.skipped, rel)
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if matchGlobs(rel, o.excludeGlobs) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		res.files = append(res.files, walkedFile{abs: p, rel: rel, size: info.Size()})
		return nil
	})
	return res, err
}

// isSVNStandardParent reports whether dir follows the trunk/branches/tags layout.
func isSVNStandardParent(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "trunk"))
	return err == nil && st.IsDir()
}

func matchGlobs(rel string, globs []string) bool {
	for _, g := range globs {
		if ok, _ := path.Match(g, rel); ok {
			return true
		}
		// allow "**/name" style and bare names
		if ok, _ := path.Match(strings.TrimPrefix(g, "**/"), path.Base(strings.TrimSuffix(rel, "/"))); ok {
			return true
		}
	}
	return false
}

// decode returns UTF-8 text. It handles UTF-8 (with/without BOM), UTF-16 with BOM,
// and falls back to Windows-1252 (superset of ISO-8859-1) for legacy sources.
func decode(b []byte) (text string, encoding string) {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		b = b[3:]
		if utf8.Valid(b) {
			return string(b), "utf-8-bom"
		}
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return decodeUTF16(b[2:], false), "utf-16le"
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return decodeUTF16(b[2:], true), "utf-16be"
	}
	if utf8.Valid(b) {
		return string(b), "utf-8"
	}
	return decodeCP1252(b), "windows-1252"
}

func decodeUTF16(b []byte, be bool) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		if be {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			u[i] = uint16(b[2*i+1])<<8 | uint16(b[2*i])
		}
	}
	return string(utf16.Decode(u))
}

var cp1252 = [32]rune{
	0x20AC, 0xFFFD, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, 0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0xFFFD, 0x017D, 0xFFFD,
	0xFFFD, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, 0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0xFFFD, 0x017E, 0x0178,
}

func decodeCP1252(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b) + len(b)/8)
	for _, c := range b {
		switch {
		case c < 0x80:
			sb.WriteByte(c)
		case c < 0xA0:
			sb.WriteRune(cp1252[c-0x80])
		default:
			sb.WriteRune(rune(c))
		}
	}
	return sb.String()
}

// isBinary uses the classic NUL-byte heuristic on the first 8 KiB.
func isBinary(b []byte) bool {
	if bytes.HasPrefix(b, []byte{0xFF, 0xFE}) || bytes.HasPrefix(b, []byte{0xFE, 0xFF}) {
		return false
	}
	n := len(b)
	if n > 8192 {
		n = 8192
	}
	return bytes.IndexByte(b[:n], 0) >= 0
}

// detectVCS inspects metadata without invoking git/svn binaries.
func detectVCS(root string) (vcs, branch, rev string) {
	if st, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		vcs = "git"
		gitDir := filepath.Join(root, ".git")
		if !st.IsDir() { // worktree / submodule: "gitdir: path"
			if b, err := os.ReadFile(gitDir); err == nil {
				p := strings.TrimSpace(strings.TrimPrefix(string(b), "gitdir:"))
				if !filepath.IsAbs(p) {
					p = filepath.Join(root, p)
				}
				gitDir = p
			}
		}
		head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
		if err != nil {
			return
		}
		h := strings.TrimSpace(string(head))
		if !strings.HasPrefix(h, "ref:") {
			return vcs, "", short(h)
		}
		ref := strings.TrimSpace(strings.TrimPrefix(h, "ref:"))
		branch = strings.TrimPrefix(ref, "refs/heads/")
		if b, err := os.ReadFile(filepath.Join(gitDir, filepath.FromSlash(ref))); err == nil {
			return vcs, branch, short(strings.TrimSpace(string(b)))
		}
		if b, err := os.ReadFile(filepath.Join(gitDir, "packed-refs")); err == nil {
			for _, l := range strings.Split(string(b), "\n") {
				if f := strings.Fields(l); len(f) == 2 && f[1] == ref {
					return vcs, branch, short(f[0])
				}
			}
		}
		return
	}
	if _, err := os.Stat(filepath.Join(root, ".svn")); err == nil {
		vcs = "svn"
		// SVN < 1.7 kept a text "entries" file with the revision on line 4.
		if b, err := os.ReadFile(filepath.Join(root, ".svn", "entries")); err == nil {
			lines := strings.Split(string(b), "\n")
			if len(lines) > 3 && strings.TrimSpace(lines[0]) != "12" {
				rev = "r" + strings.TrimSpace(lines[3])
			}
		}
		return
	}
	if _, err := os.Stat(filepath.Join(root, "CVS")); err == nil {
		return "cvs", "", ""
	}
	return "none", "", ""
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
