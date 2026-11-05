package scan

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/build"
	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/model"
)

// Built artifacts (WAR/EAR) of modules that ship no code are read from the
// inside: descriptors, configuration and embedded libraries. Nothing is
// extracted to disk, no class is loaded and nothing is executed. Entries are
// named "<artifact>!/<entry>" so every finding points to its origin.

const (
	maxArchiveEntries = 100_000
	maxNestedBytes    = 64 << 20  // nested JAR/WAR read in memory
	maxArchiveBudget  = 512 << 20 // total bytes decompressed per artifact
)

var reArtifactVersion = regexp.MustCompile(`^(.+?)[-_]v?(\d+(?:[.\-_][0-9A-Za-z]+)*)\.(war|ear|jar)$`)

// ArtifactVersion extracts the version from a built artifact name
// (sign-gateway-2.4-INT.war → "2.4-INT").
func ArtifactVersion(name string) string {
	if m := reArtifactVersion.FindStringSubmatch(path.Base(name)); m != nil {
		return m[2]
	}
	return ""
}

// textEntry selects the entries worth analyzing: descriptors and configuration.
func textEntry(name string) bool {
	n := strings.ToLower(name)
	if strings.HasSuffix(n, "/") || strings.Contains(n, "/lib/") || strings.HasPrefix(n, "meta-inf/maven/") {
		return false // embedded build metadata: the module is already known from the repository
	}
	ext := path.Ext(n)
	switch ext {
	case ".xml", ".properties", ".yml", ".yaml", ".wsdl":
	default:
		return false
	}
	return strings.HasPrefix(n, "web-inf/") || strings.HasPrefix(n, "meta-inf/") || !strings.Contains(n, "/")
}

func introspectArchive(r *fileResult, abs, rel string, size int64, o Options) {
	zr, err := zip.OpenReader(abs)
	if err != nil {
		r.diag = append(r.diag, model.Diagnostic{File: rel, Kind: "artifact-unreadable", Message: err.Error()})
		return
	}
	defer zr.Close()
	budget := int64(maxArchiveBudget)
	readArchive(r, &zr.Reader, rel, o, &budget, 0)
}

func readArchive(r *fileResult, zr *zip.Reader, rel string, o Options, budget *int64, depth int) {
	if len(zr.File) > maxArchiveEntries {
		r.diag = append(r.diag, model.Diagnostic{File: rel, Kind: "artifact-too-big", Message: fmt.Sprintf("%d entries: not read", len(zr.File))})
		return
	}
	classes := 0
	for _, f := range zr.File {
		name := f.Name
		low := strings.ToLower(name)
		inner := rel + "!/" + name
		switch {
		case strings.HasSuffix(low, ".class"):
			classes++
		case strings.HasSuffix(low, ".jar") && (strings.HasPrefix(low, "web-inf/lib/") || strings.HasPrefix(low, "lib/") || depth == 0 && !strings.Contains(low, "/")):
			b, ok := readEntry(f, budget, maxNestedBytes)
			if !ok {
				d := build.VendoredJar(inner)
				r.libs = append(r.libs, d)
				continue
			}
			nz, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
			if err != nil {
				r.libs = append(r.libs, build.VendoredJar(inner))
				continue
			}
			r.libs = append(r.libs, build.VendoredJarFrom(nz, inner))
		case (strings.HasSuffix(low, ".war") || strings.HasSuffix(low, ".jar") && strings.Contains(low, "ejb")) && depth == 0:
			// EAR module: one more level
			b, ok := readEntry(f, budget, maxNestedBytes)
			if !ok {
				continue
			}
			if nz, err := zip.NewReader(bytes.NewReader(b), int64(len(b))); err == nil {
				readArchive(r, nz, inner, o, budget, depth+1)
			}
		case strings.HasPrefix(low, "meta-inf/maven/") && strings.HasSuffix(low, "/pom.properties") && depth == 0:
			// the artifact's own coordinates are read by the caller through build.ZipCoordinates
		case textEntry(name):
			if int64(f.UncompressedSize64) > o.MaxFileSize {
				continue
			}
			b, ok := readEntry(f, budget, o.MaxFileSize)
			if !ok {
				continue
			}
			l, known := lang.Detect(name)
			if !known {
				continue
			}
			ir := &fileResult{isInner: true, file: model.File{Path: inner, Language: l.Name, Category: l.Category,
				Bytes: int64(len(b)), SourceSet: "main"}}
			analyzeText(ir, inner, b, l, o)
			r.inner = append(r.inner, ir)
		}
	}
	if depth == 0 {
		named := ArtifactVersion(rel)
		if g, a, v, ok := build.ZipCoordinates(zr, rel); ok {
			r.artifact = &artifactInfo{rel: rel, coord: "maven:" + g + ":" + a, version: v, classes: classes}
			if named != "" && v != "" && named != v && !strings.HasPrefix(named, v+"-") {
				r.diag = append(r.diag, model.Diagnostic{File: rel, Kind: "artifact-name-version",
					Message: fmt.Sprintf("the file name says version %s but the build metadata inside the artifact says %s", named, v)})
			}
		} else {
			r.artifact = &artifactInfo{rel: rel, version: named, classes: classes}
		}
	}
}

type artifactInfo struct {
	rel, coord, version string
	classes             int
}

func readEntry(f *zip.File, budget *int64, limit int64) ([]byte, bool) {
	if int64(f.UncompressedSize64) > limit || int64(f.UncompressedSize64) > *budget {
		return nil, false
	}
	rc, err := f.Open()
	if err != nil {
		return nil, false
	}
	defer rc.Close()
	b, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, false
	}
	*budget -= int64(len(b))
	return b, true
}
