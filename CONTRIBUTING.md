# Contributing to Traverse

Thanks for helping! Traverse aims to stay **dependency-free, offline and deterministic**. Please keep that in mind.

## Development

```sh
go test ./...                 # unit + regression tests (fixtures in internal/scan/testdata)
go run ./cmd/traverse rules test   # verify all rule examples
gofmt -l .                    # must print nothing
go vet ./...
```

## Architecture

```
cmd/traverse            CLI
internal/survey         structure reconnaissance: where the code is, copies, main line
internal/scan           repository walk, file analysis, config URLs, archives, keystores, certificates, classification
internal/lang           language detection + comment/string lexer (offsets preserved)
internal/java           structural Java parser (types, annotations, members)
internal/lite           module-level structure for JS/TS, Python, PHP, Go, Ruby
internal/legacy         COBOL, JCL, BMS, CICS CSD
internal/descriptors    JEE / JBoss / Spring / Struts / WSDL / server descriptors
internal/build          build manifests and lockfiles (Maven, Gradle, Ant, Ivy, npm, MSBuild, ...)
internal/deploy         Dockerfile, Compose, Kubernetes, OpenShift, Helm
internal/sqlparse       table references in SQL text
internal/yamlite        dependency-free YAML subset reader
internal/rules          data-driven rule engine + embedded catalog (default.json)
internal/vuln           offline OSV matching
internal/secure         host sealing (HMAC pseudonyms + AES-GCM vault)
internal/portfolio      cross-repository aggregation (links, shared tables, libraries)
internal/report         HTML, Markdown, CSV, SVG, Mermaid, Structurizr, PlantUML, ArchiMate, SARIF, CycloneDX
internal/model          the facts schema: the stable contract between stages
rulepacks/              optional regional / sector rule packs (not embedded)
```

The **facts schema** (`internal/model`) is the public contract. Additive changes bump MINOR in `model.SchemaVersion`; breaking changes bump MAJOR.

## Ground rules

1. **No third-party modules.** If you think one is unavoidable, open an issue first.
2. **No network, no process execution, no writes outside `--out`.**
3. **Deterministic output:** sort everything you emit; no map iteration order in outputs.
4. **Never copy code lines, property values or secrets into outputs.**
5. Every behaviour change comes with a test; every rule with examples.
6. Spec-driven: non-trivial changes start as an OpenSpec change under `openspec/changes/`.

## Adding a rule

See [docs/rules.md](docs/rules.md). Edit `tools/genrules.py` and regenerate the catalog and the packs with `python3 tools/genrules.py internal/rules/default.json rulepacks`, then run `go run ./cmd/traverse rules test`. The core catalog stays agnostic: country-, sector- or organization-specific rules go into a pack under `rulepacks/`.

## Adding a language or extractor

- New language: register it in `internal/lang/languages.go` with its comment/string syntax.
- New build system: add a parser in `internal/build` returning a `model.Module` with a stable `ID` scheme (`<ecosystem>:<coordinates>`) so the portfolio can link producers and consumers.
- New contract source: emit `model.Contract` with a normalized `Match` key; `internal/portfolio` links contracts of the same `Kind`.

## Commit and PR

Conventional commits are appreciated (`feat:`, `fix:`, `rules:`, `docs:`). Include before/after output for behaviour changes on `examples/demo-portfolio`.
