package model

import (
	"regexp"
	"strings"
)

var jndiPrefixes = []string{
	"java:comp/env/", "java:jboss/exported/", "java:jboss/", "java:global/", "java:app/",
	"java:module/", "java:/", "java:",
}

// NormalizeJNDI strips namespace prefixes so that bindings declared in
// different places (web.xml, @Resource, server config) can be compared.
func NormalizeJNDI(name string) string {
	n := strings.TrimSpace(name)
	for _, p := range jndiPrefixes {
		if strings.HasPrefix(n, p) {
			n = n[len(p):]
			break
		}
	}
	return strings.Trim(n, "/")
}

var reVar = regexp.MustCompile(`\{[^}]*\}|:[A-Za-z_]\w*`)
var reSlashes = regexp.MustCompile(`/+`)

// NormalizePath canonicalizes an HTTP path for matching: path variables
// become {}, duplicate and trailing slashes are removed.
func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	p = reVar.ReplaceAllString(p, "{}")
	p = reSlashes.ReplaceAllString("/"+p, "/")
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// JoinPath joins URL path segments.
func JoinPath(parts ...string) string {
	var segs []string
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), "/")
		if p != "" && p != "*" {
			segs = append(segs, p)
		}
	}
	return "/" + strings.Join(segs, "/")
}

// LiteralSegments counts path segments that are neither variables nor wildcards.
func LiteralSegments(p string) int {
	n := 0
	for _, s := range strings.Split(p, "/") {
		if s != "" && s != "{}" && !strings.Contains(s, "*") {
			n++
		}
	}
	return n
}

// URLTransport classifies a URL's transport by scheme: tls for https/wss and
// other TLS schemes, plain for http/ws/ftp, and "" when unknown.
func URLTransport(u string) string {
	i := indexScheme(u)
	if i < 0 {
		return ""
	}
	switch toLowerASCII(u[:i]) {
	case "https", "wss", "ldaps", "ftps", "sftp", "smtps", "imaps", "pop3s":
		return "tls"
	case "http", "ws", "ftp", "ldap", "smtp", "imap", "pop3", "telnet", "jdbc":
		return "plain"
	}
	return ""
}

func indexScheme(u string) int {
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c == ':' {
			return i
		}
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
			return -1
		}
	}
	return -1
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}
