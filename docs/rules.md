# Writing detection rules

Rules are JSON objects evaluated on **comment-free** source text (string literals preserved), so commented-out code never matches. Patterns use Go RE2 syntax (no backtracking: linear time, safe on any input).

```json
{
  "id": "JAVA-AXIS1",
  "title": "Apache Axis 1 (end of life)",
  "category": "Integration",
  "technology": "Axis 1",
  "severity": "high",
  "confidence": "high",
  "languages": ["Java", "XML"],
  "pattern": "\\borg\\.apache\\.axis\\.[a-z]",
  "message": "Unmaintained since 2006 with known vulnerabilities.",
  "tags": ["eol", "security"],
  "examples": {
    "match":   ["import org.apache.axis.client.Call;"],
    "noMatch": ["import org.apache.axis2.client.ServiceClient;"]
  }
}
```

| Field | Meaning |
|---|---|
| `id` | Unique, stable. A rule file with an existing id overrides the built-in rule |
| `severity` | `info` (technology inventory), `low`, `medium`, `high` (risks) |
| `confidence` | `low`, `medium`, `high`: how likely a match is a true positive |
| `languages` | Language names as in `internal/lang/languages.go`; empty = all text files |
| `pattern` | Content regex. One finding per (rule, file) with match count and lines |
| `pathPattern` | Regex on the relative path. Alone: file-presence rule. With `pattern`: restricts files |
| `exclude` | Optional regex applied to the matched text: a match that also matches it is dropped (e.g. a URL where a password was expected) |
| (id prefix) `TLS-UPSTREAM-` | A rule whose id starts with this prefix marks the inbound endpoints of the matching file (or, for filters and interceptors, of its module) as *mTLS at an intermediary (inferred)* |
| `authKind` | Optional, for authentication rules: `user` (login), `service` (service to service), `signature` (e-signature) or `transport` (TLS/mTLS). Decides where the mechanism appears in the security section; empty = derived from the category |
| `examples` | **Required**. `noMatch` examples are lexed with the first language's syntax, so they can prove that comments are ignored |

Tips:

- Anchor on fully qualified package names (`\borg\.apache\.struts\.action\b`) rather than simple class names.
- Prefer distinctive literals: the engine derives a literal prefilter from the pattern and skips files that cannot match, which keeps scans fast.
- Secret rules must report location only; never design a rule whose purpose is to extract values.

Test:

```sh
traverse rules test --rules my-rules.json        # built-in + yours
traverse rules test --no-default-rules --rules my-rules.json
```

## Core catalog and rule packs

The embedded catalog (`internal/rules/default.json`) is **agnostic**: it recognizes standards,
open-source frameworks and vendor products used anywhere (Spring Security, OAuth2, SAML, eIDAS,
XML-DSig, XAdES/CAdES/PAdES, RFC 3161, PKCS#11…), never the platforms of one country, sector or
organization. That knowledge goes into **rule packs**: ordinary rule files kept in
[`rulepacks/`](../rulepacks/), loaded only when asked for.

```sh
traverse scan --rules rulepacks/es-public-sector.json --out reports app/
```

or once per portfolio, in `traverse.json` (paths relative to the file):

```json
{ "rules": ["../rulepacks/es-public-sector.json"] }
```

Pack conventions: ids prefixed with the scope (`ES-`, `ACME-`…) so they never override a core
rule by accident (a test enforces it), a `version` such as `es-public-sector-1.0.0`, and the same
examples requirement as any rule. `tools/genrules.py` regenerates the core catalog and the packs.
