package build

import (
	"archive/zip"
	"bufio"
	"io"
	"path"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// Limits for reading metadata inside committed archives.
const (
	maxJarEntries   = 200_000
	maxPropsBytes   = 64 << 10
	maxJarFileBytes = 512 << 20
)

// JarCoordinates reads META-INF/maven/<g>/<a>/pom.properties inside a JAR
// (read-only; nothing is extracted or executed). For shaded JARs with several
// descriptors it picks the one whose artifactId matches the file name, or the
// only one present. ok is false when no usable coordinates are found.
func JarCoordinates(absPath, rel string) (group, artifact, version string, ok bool) {
	zr, err := zip.OpenReader(absPath)
	if err != nil {
		return
	}
	defer zr.Close()
	return ZipCoordinates(&zr.Reader, rel)
}

// ZipCoordinates is JarCoordinates on an archive already open (for instance a
// JAR nested in a WAR, read from memory).
func ZipCoordinates(zr *zip.Reader, rel string) (group, artifact, version string, ok bool) {
	if len(zr.File) > maxJarEntries {
		return
	}
	base := strings.ToLower(strings.TrimSuffix(path.Base(rel), path.Ext(rel)))
	type coord struct{ g, a, v string }
	var found []coord
	for _, f := range zr.File {
		n := f.Name
		if !strings.HasPrefix(n, "META-INF/maven/") || !strings.HasSuffix(n, "/pom.properties") || f.UncompressedSize64 > maxPropsBytes {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		c := coord{}
		sc := bufio.NewScanner(io.LimitReader(rc, maxPropsBytes))
		for sc.Scan() {
			k, v, has := strings.Cut(strings.TrimSpace(sc.Text()), "=")
			if !has || strings.HasPrefix(k, "#") {
				continue
			}
			switch strings.TrimSpace(k) {
			case "groupId":
				c.g = strings.TrimSpace(v)
			case "artifactId":
				c.a = strings.TrimSpace(v)
			case "version":
				c.v = strings.TrimSpace(v)
			}
		}
		rc.Close()
		if c.g != "" && c.a != "" && c.v != "" {
			found = append(found, c)
		}
	}
	for _, c := range found {
		if strings.HasPrefix(base, strings.ToLower(c.a)) {
			return c.g, c.a, c.v, true
		}
	}
	if len(found) == 1 {
		return found[0].g, found[0].a, found[0].v, true
	}
	return
}

// VendoredJarFrom builds the dependency of a JAR read from an open archive.
func VendoredJarFrom(zr *zip.Reader, rel string) model.Dependency {
	d := VendoredJar(rel)
	if g, a, v, ok := ZipCoordinates(zr, rel); ok {
		d.ID, d.Version, d.Resolved = "maven:"+g+":"+a, v, true
	}
	return d
}

// VendoredJarAt is VendoredJar enriched with the Maven coordinates embedded in
// the archive, when present.
func VendoredJarAt(absPath, rel string, size int64) model.Dependency {
	d := VendoredJar(rel)
	if size > maxJarFileBytes {
		return d
	}
	if g, a, v, ok := JarCoordinates(absPath, rel); ok {
		d.ID, d.Version, d.Resolved = "maven:"+g+":"+a, v, true
	}
	return d
}
