# Traverse – Offline Architecture Analyzer

**Offline, read-only static architecture analysis for portfolios of (legacy) applications.**
No AI, no network, no build tools, no installation: one self-contained executable for Windows, Linux and macOS.

Traverse reads source code checked out from Git, Subversion or anywhere else and produces:

- **Per-application reports**: system context, layered architecture, exposed and consumed interfaces (REST, SOAP, JMS, Kafka, JNDI, EJB), database tables, build modules, deployment units, technologies, risks, package coupling, cycles and layer violations.
- **A portfolio view** across repositories: which system calls which, shared queues, **tables shared by several applications**, internal libraries and **version divergence**, and dependencies that point outside the portfolio.
- **Known vulnerabilities, offline**: library versions (declared, pinned by lockfiles — transitives included — or embedded in committed JARs) matched against an [OSV](https://osv.dev) dump you download beforehand. Nothing is sent anywhere.
- **Authentication, mTLS and third-party inventory**: detects authentication mechanisms (container, Spring Security, OAuth2/JWT, SAML, CAS, Keycloak, Kerberos, LDAP, WS-Security, API keys), mutual TLS (client certificates in web.xml, Tomcat, WildFly, Spring Boot, client keystores, and **mTLS terminated by a proxy or API gateway, inferred per endpoint from client-certificate headers**), transport per integration (TLS vs plaintext), electronic signature (XML-DSig, CMS/PKCS#7, PDF, XAdES/CAdES/PAdES, RFC 3161 timestamping, PKCS#11 tokens) and federated identity (eIDAS), and lists every third-party system the application talks to.
- **Structure reconnaissance first** (`traverse survey`): before reading any code it finds where the code is, which folders are copies of the same project (release copies, SVN branches/tags, "entregas/<version>", trunk2…), picks the main line and sets aside documentation, models, bundled runtimes and binaries. Modules shipped only as WAR/EAR are read from the inside (descriptors, configuration, embedded libraries).
- **Integrations from configuration and objective counterparts**: URLs in `.properties`, YAML, Spring XML and server descriptors become integrations; counterparts are named from the code itself (client package, config key, JNDI name, foreign DB schema, platform module). API endpoints are told apart from user screens, and authentication is split into user login, service-to-service, e-signature and transport.
- **Executive summary and Markdown export**: every report carries a short executive summary, and the full analysis is exported to a populated `report.md` (plus arc42.md), not just HTML.
- **Reusable exports**: CSV (spreadsheet-ready), JSON, SVG, Mermaid, **Structurizr DSL (C4 model)** and **C4-PlantUML**.

Every statement in a report is traceable to `file:line` and a versioned rule.

## Coverage

| Level | Technologies |
|---|---|
| Structural (types, layers, dependencies, endpoints, contracts) | **Java** (1.2 → 21+), **Kotlin**, **C#** (.NET Framework and .NET 5+: attribute and conventional routing, minimal APIs, Razor Pages) |
| Modules, imports, HTTP routes, ORM tables | **JS/TS** (Express, NestJS, TypeORM, Sequelize, Knex), **Python** (Django, Flask, FastAPI, SQLAlchemy), **PHP** (Laravel, Symfony, Eloquent), **Go** (gin, echo, chi, net/http, GORM), **Ruby** (Rails) |
| Mainframe | **COBOL** (programs, copybooks, CALL, CICS LINK/XCTL incl. MOVE-resolved targets, BMS maps, files, TS/TD queues, EXEC SQL), **JCL** (jobs, steps, procs, datasets with DISP), **BMS**, **CICS CSD** (transactions, file→dataset), HLASM |
| Inventory, technologies and risks | 80 languages and formats incl. PL/I, RPG, Natural, VB6, Delphi, PowerBuilder, PL/SQL, REXX, ASP, JSP |
| Build manifests | Maven (parent inheritance, properties, dependencyManagement, finalName), Ant, Ivy, Gradle, npm, Composer, MSBuild/NuGet (incl. central package management), Python, Go, vendored JARs (coordinates read from embedded `pom.properties`) |
| Lockfiles | package-lock.json, npm-shrinkwrap.json, yarn.lock (v1 and Berry), pnpm-lock.yaml, bun.lock, composer.lock, packages.lock.json, poetry.lock, uv.lock, Pipfile.lock, gradle.lockfile |
| Vulnerabilities | Offline OSV dumps (Maven, npm, NuGet, PyPI, Packagist, Go); ecosystem-specific version ordering (Maven ComparableVersion, SemVer, NuGet, PEP 440); CVSS v3.x base score; malicious packages; aliases unified (GHSA/CVE/PYSEC) |
| Descriptors | web.xml, jboss-web.xml, jboss-deployment-structure.xml, application.xml, ejb-jar.xml, persistence.xml, Spring XML, Struts 1/2, JSF, WSDL, standalone.xml, *-ds.xml, *-jms.xml |
| Rules | 165 data-driven rules (frameworks, platforms, EOL technologies, javax→jakarta, JDK removals, .NET Framework, Python 2, secrets location, DB vendors…), plus optional regional [rule packs](rulepacks/) |
| Version control | Git metadata (branch/commit); Subversion `trunk/branches/tags` layout (only `trunk` by default) |
| Deployment (IaC) | Dockerfile, Docker Compose (incl. overrides), Kubernetes, OpenShift (DeploymentConfig, Route), Helm templates — own YAML reader, no external tools |
| Documentation | arc42 per system and a portfolio document in Markdown with Mermaid diagrams and a full **integration catalog** (direction, counterpart, mode, evidence, confidence) |
| Encodings | UTF-8, UTF-16, Windows-1252/ISO-8859-1 fallback |

## Standard outputs

All validated in CI-style checks against the official schema or tool:

| Output | Standard | Validated with |
|---|---|---|
| `workspace.dsl` | C4 model / Structurizr DSL (context, containers, components, deployment) | Structurizr CLI `validate` + `export` |
| `landscape.puml` | C4-PlantUML | PlantUML (bundled C4 stdlib) |
| `architecture.archimate.xml` | ArchiMate 3.1 Model Exchange File Format (The Open Group) | Official XSD (`archimate3_Diagram.xsd`) |
| `findings.sarif` | SARIF 2.1.0 (OASIS) | Official JSON schema |
| `sbom.cdx.json` | CycloneDX 1.6 (ECMA-424) with Package URLs and, with `--vulndb`, `vulnerabilities` | Official JSON schema |
| `*.mmd`, Mermaid blocks in `arc42.md` / `portfolio.md` | Mermaid | Mermaid CLI |

## Quick start

Traverse is built from source (Go ≥ 1.24, standard library only: no module downloads, works offline).

**Linux / macOS**

```sh
git clone https://github.com/oskillarrr/traverse.git && cd traverse
go build -ldflags "-X main.version=0.9.4" -o traverse ./cmd/traverse
./traverse scan --manifest examples/repos.csv --out ../traverse-demo     # try the bundled demo

./traverse keygen --out ~/keys/team.key            # optional: encrypted host names; keep it out of repos and reports
./traverse survey /src/my-app                      # where the code is, copies and main line (reads no code)
./traverse scan --manifest repos.csv --out ~/reports/portfolio --key-file ~/keys/team.key
./traverse scan --manifest repos.csv --out ~/reports/portfolio --vulndb osv/2026-11-03   # + offline OSV dump
```

**Windows (PowerShell)**

```powershell
git clone https://github.com/oskillarrr/traverse.git; cd traverse
go build -ldflags "-X main.version=0.9.4" -o traverse.exe .\cmd\traverse
.\traverse.exe scan --manifest examples\repos.csv --out ..\traverse-demo
.\traverse.exe keygen --out $HOME\keys\team.key
.\traverse.exe scan --manifest repos.csv --out $HOME\reports\portfolio --key-file $HOME\keys\team.key
```

- `repos.csv` has one line per system: `name,path,group` ([example](examples/demo-portfolio/repos.csv)).
- The core rules are country-agnostic; regional knowledge is opt-in through [rule packs](rulepacks/) (`--rules rulepacks/<pack>.json`).
- `sh scripts/build.sh 0.9.4` / `scripts\build.ps1 -Version 0.9.4` build all platforms into `dist/` with `SHA256SUMS.txt`.

**User guide: [docs/manual.md](docs/manual.md)** ([español](docs/manual.es.md)) · Sample output: [examples/demo-report/index.html](examples/demo-report/index.html) · Rule authoring and rule packs: [docs/rules.md](docs/rules.md) · Design and specs: [openspec/](openspec/)

## Security model

Read-only on analyzed code · writes only to `--out` (refused inside a repository) · no network code, XML DTDs never resolved · vulnerability data read from local files only · archives (JARs, OSV zips) read in place with size limits, never extracted or executed · never executes processes · no telemetry · Go standard library only.
Reports never contain code lines, property or environment values, secrets, full URLs, clear-text host names or the subject of personal certificates. With a key (`traverse keygen`), host names are kept as HMAC pseudonyms and stored AES-256-GCM-encrypted in `hosts.vault.json` (`traverse reveal` to decrypt). See [SECURITY.md](SECURITY.md).

## Limitations

Static and heuristic: relationships are evidence-based candidates, not runtime facts. Reflection, dynamic configuration and anything outside the repository (server configuration, load balancers, ESB) are invisible unless provided as input. Validate findings before using them in official architecture documents.

## Contributing

Rules are JSON with mandatory positive and negative examples; new extractors plug into a stable facts schema. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
