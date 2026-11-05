# Traverse user guide

> Version 0.9.4 · Offline, read-only static architecture analysis. ·
> [Versión en español](manual.es.md)

Traverse reads the source code of one or many applications and tells you, with `file:line`
evidence, what each application is made of, what it exposes and calls, which data it shares, how
it authenticates and which risks it carries. It runs on **Linux, macOS and Windows**, needs no
installation, never connects to the network and never executes the code it analyzes.

## Contents

0. [Getting started](#0-getting-started)
1. [What Traverse is and is not](#1-what-traverse-is-and-is-not)
2. [Get the executable](#2-get-the-executable)
3. [Prepare the code](#3-prepare-the-code)
4. [Run an analysis](#4-run-an-analysis)
5. [Read the results](#5-read-the-results)
6. [Validate the findings](#6-validate-the-findings)
7. [Configuration, custom rules and rule packs](#7-configuration-custom-rules-and-rule-packs)
8. [Security and privacy](#8-security-and-privacy)
9. [Limitations](#9-limitations)
10. [Troubleshooting](#10-troubleshooting)
11. [Appendix: locked-down machines](#11-appendix-locked-down-machines)

Commands are given for **Linux/macOS (sh)** and **Windows (PowerShell)**. They are equivalent.

---

## 0. Getting started

About fifteen minutes from nothing to a portfolio report. You only need **Go 1.24 or later** to
build the executable ([section 2.2](#22-build-from-source)); the analysis itself needs nothing.

**1. Build.** Stamping the version makes every report say which version produced it.

```sh
# Linux / macOS
git clone https://github.com/oskillarrr/traverse.git && cd traverse
go build -ldflags "-X main.version=0.9.4" -o traverse ./cmd/traverse
./traverse version                        # traverse 0.9.4 schema 1.5.0
T="$PWD/traverse"
```

```powershell
# Windows
git clone https://github.com/oskillarrr/traverse.git; cd traverse
go build -ldflags "-X main.version=0.9.4" -o traverse.exe .\cmd\traverse
.\traverse.exe version                    # traverse 0.9.4 schema 1.5.0
$T = (Resolve-Path .\traverse.exe).Path
```

**2. Try it on the bundled demo** (five fictional systems, no setup):

```sh
"$T" scan --manifest examples/repos.csv --out ../traverse-demo
```

```powershell
& $T scan --manifest examples\repos.csv --out ..\traverse-demo
```

Open `../traverse-demo/index.html` in any browser. A pre-generated copy is in
[examples/demo-report](../examples/demo-report/index.html).

**3. A key for host names (recommended).** Host names found in the code are infrastructure data.
With a key they are replaced by stable pseudonyms and stored encrypted; without a key they are not
recorded at all. If several people analyze the same portfolio, **share one key** so pseudonyms
match across reports. Keep it outside repositories and report folders.

```sh
"$T" keygen --out ~/keys/traverse-team.key
```

```powershell
& $T keygen --out $HOME\keys\traverse-team.key
```

**4. A manifest of your applications.** One line per system: `name,path,group`. `group` (an area,
domain or team) is optional and produces one view per group. If an application has several lines of
code and you want a specific one, point `path` at it.

```csv
name,path,group
billing,/src/billing,Finance
ledger,/src/ledger/trunk,Finance
web-portal,/src/portal,Channels
```

Paths may be absolute or relative to the CSV. On Windows use `C:\src\billing`.

**5. Check where the code is.** `survey` reads only folder names, sizes and build descriptors, and
takes seconds. Make sure the **main line** it picks is the one you expect.

```sh
tail -n +2 repos.csv | cut -d, -f2 | while read -r p; do "$T" survey "$p"; done
```

```powershell
Import-Csv repos.csv | ForEach-Object { "===== $($_.name)"; & $T survey $_.path }
```

**6. Analyze everything together and open the report.** Analyzing all applications in one run is
what reveals the relationships between them.

```sh
"$T" scan --manifest repos.csv --out ~/reports/portfolio --key-file ~/keys/traverse-team.key
xdg-open ~/reports/portfolio/index.html      # macOS: open ...
```

```powershell
& $T scan --manifest repos.csv --out $HOME\reports\portfolio --key-file $HOME\keys\traverse-team.key
Start-Process $HOME\reports\portfolio\index.html
```

Add `--lang es` for reports in Spanish. Add a rule pack with `--rules` if one exists for your
country or sector ([section 7.3](#73-rule-packs)).

**7. Review.** Start with the executive summary and, for each system, **Relationships with other
applications**. Then ask someone who knows the applications to fill in `validation-sample.csv`
([section 5.10](#510-measure-reliability-validation-samplecsv)). Share the report folder, never the
key.

---

## 1. What Traverse is and is not

Traverse produces, **without AI, without network access and without building or running the
analyzed code**:

- **A report per application**: system context, layered architecture, exposed and consumed
  interfaces (REST, SOAP, JMS, Kafka, JNDI, EJB), database tables, build modules, deployment units,
  technologies, risks, components, coupling metrics, cycles and layer violations.
- **A portfolio report**: which system calls which, shared queues, tables used by several
  applications, internal libraries and their version divergence, and dependencies pointing outside
  the portfolio.
- **Authentication and transport**: user login, service-to-service authentication, electronic
  signature, TLS and mutual TLS, certificates, and an inventory of third-party systems.
- **Known vulnerabilities, offline**, from an [OSV](https://osv.dev) dump you download beforehand.
- **Exports in standard formats**, validated against their official schemas or tools:

| Format | Standard | Use |
|---|---|---|
| Structurizr DSL | C4 model | Context, container, component and deployment diagrams |
| C4-PlantUML | PlantUML + C4 library | Diagrams in technical documentation |
| ArchiMate 3.1 Model Exchange File | The Open Group | Import into Archi, Sparx EA, BiZZdesign, ADOIT… |
| SARIF 2.1.0 | OASIS | Findings in GitLab, GitHub, SonarQube, DefectDojo, VS Code… |
| CycloneDX 1.6 | ECMA-424 (OWASP) | SBOM with Package URLs (and vulnerabilities with `--vulndb`) |
| Markdown (arc42, report), Mermaid, SVG, CSV, JSON | — | Wikis, spreadsheets, integration |

**Coverage:**

| Level | Technologies |
|---|---|
| Structural (components, layers, dependencies, endpoints, contracts) | **Java** (1.2 → 21+), **Kotlin**, **C#** (.NET Framework and .NET 5+) |
| Modules, imports, HTTP routes, ORM tables | **JavaScript/TypeScript**, **Python**, **PHP**, **Go**, **Ruby** |
| Mainframe | **COBOL**, **JCL**, **BMS**, **CICS CSD**, HLASM |
| Server pages | JSP, ASP.NET Web Forms/ASMX/WCF, Razor Pages, classic ASP |
| Inventory, technologies and risks | 80 languages and formats (PL/I, RPG, Natural, VB6, Delphi, PowerBuilder, PL/SQL, REXX…) |
| Build | Maven, Ant, Ivy, Gradle, npm, Composer, MSBuild/NuGet, Python, Go, JARs committed to the repository, WAR/EAR read from the inside |

The embedded catalog has **165 rules** for technologies and risks. It is **agnostic**: it knows
standards, open-source frameworks and products used anywhere, never the platforms of one country or
organization. That knowledge goes into optional [rule packs](#73-rule-packs).

**Traverse is not** a substitute for the knowledge of the teams or for the real production
configuration. Its relationships are *candidates with evidence*, not runtime facts. See
[Limitations](#9-limitations).

---

## 2. Get the executable

Traverse is written in Go, but **Go is only needed to build it**. The result is one self-contained
executable: no installer, no runtime, no administrator rights.

### 2.1 Pick a route

| Route | When |
|---|---|
| **Build from source** (2.2) | Default. Works offline: the project has no third-party modules |
| **Build in your organization's CI** (2.4) | You want binaries produced and, if required, signed by your own infrastructure |
| **Binaries handed to you** (2.5) | Someone already built them; verify the checksums |

| Platform | Built file name |
|---|---|
| Linux x64 / ARM | `traverse-<version>-linux-amd64` / `-linux-arm64` |
| macOS Intel / Apple Silicon | `traverse-<version>-darwin-amd64` / `-darwin-arm64` |
| Windows x64 / ARM | `traverse-<version>-windows-amd64.exe` / `-windows-arm64.exe` |

### 2.2 Build from source

1. Install **Go 1.24+**. The archive (`.tar.gz` / `.zip`, not the installer) needs no
   administrator rights: https://go.dev/dl/ or your internal software repository.

```sh
# Linux / macOS: unpack, then for this session
export PATH="$HOME/go-sdk/bin:$PATH"; go version
```

```powershell
# Windows: unpack to e.g. C:\Tools\go, then for this session
$env:Path = "C:\Tools\go\bin;$env:Path"; go version
```

2. Get the source (`git clone` or the source archive) and build:

```sh
go test ./...                                   # optional: unit and regression tests
go run ./cmd/traverse rules test                # optional: verify every rule example
go build -ldflags "-X main.version=0.9.4" -o traverse ./cmd/traverse
```

```powershell
go build -ldflags "-X main.version=0.9.4" -o traverse.exe .\cmd\traverse
```

Without `-ldflags` the executable reports version `dev`; it works the same.

### 2.3 All platforms at once

```sh
sh scripts/build.sh 0.9.4                       # Linux / macOS / Git Bash
```

```powershell
.\scripts\build.ps1 -Version 0.9.4               # Windows PowerShell 5.1 or 7
```

Both produce `dist/` with the six executables and `SHA256SUMS.txt`. The build is reproducible:
`-trimpath`, no CGO, no local paths.

### 2.4 Build in GitLab CI or GitHub Actions

The repository includes `.gitlab-ci.yml` (test → build → release on tags) and
`.github/workflows/ci.yml`. Both run formatting, `go vet`, the tests and the rule examples, and
publish `dist/` as an artifact. If your runner has no Internet access, mirror the `golang:1.24`
image and set `GO_IMAGE`. Code signing can be added in the release stage.

### 2.5 Verify and place a binary

```sh
sha256sum traverse-0.9.4-linux-amd64            # macOS: shasum -a 256 …; compare with SHA256SUMS.txt
mkdir -p ~/bin && cp traverse-0.9.4-linux-amd64 ~/bin/traverse && chmod +x ~/bin/traverse
```

```powershell
Get-FileHash .\traverse-0.9.4-windows-amd64.exe -Algorithm SHA256
New-Item -ItemType Directory $HOME\Tools\traverse -Force
Copy-Item .\traverse-0.9.4-windows-amd64.exe $HOME\Tools\traverse\traverse.exe
```

If the machine blocks the executable, see the [appendix](#11-appendix-locked-down-machines).

---

## 3. Prepare the code

Traverse **does not connect to Git, Subversion or any server**. It analyzes folders already on
disk. Check the code out with your usual credentials, in an authorized location.

### 3.1 Git

```sh
git clone --depth 1 https://git.example.org/team/app-a.git
git clone --depth 1 --branch release/2.3 https://git.example.org/team/app-b.git
```

`--depth 1` is enough: Traverse analyzes the current state, not the history. If `.git` is present,
the report records the branch and the commit.

### 3.2 Subversion and other layouts

You can point Traverse at an application folder as it is, with documentation, models, data
scripts, branches, tags and release copies around the code. Before reading code, Traverse
recognizes the structure (3.3) and analyzes only the main line.

```sh
svn checkout https://svn.example.org/repos/app-c app-c        # whole layout: fine
svn export   https://svn.example.org/repos/app-c/trunk app-c  # or only trunk, without .svn
```

### 3.3 Structure reconnaissance (`traverse survey`)

`survey` walks the folder levels (names, sizes and build descriptors only) to find **where the
code is** and **which folders are copies of the same project**:

```
case-management/   20 files
├─ branches/   1 files
│  └─ release-1.0/   copy of trunk · 1 files
├─ tags/   1 files
│  └─ v1.0/   copy of trunk · 1 files
└─ trunk/   MAIN LINE · maven:org.example.cases:case-parent 1.4.2 · 18 files
   ├─ case-ear/   maven:org.example.cases:case-ear 1.4.2 · 2 files
   ├─ case-ejb/   maven:org.example.cases:case-ejb 1.4.2 · 7 files
   ├─ case-web/   maven:org.example.cases:case-web 1.4.2 · 7 files
   └─ db/   data (SQL) · 1 files
```

All decisions are objective:

- **Copies**: two folders are the same project if they share build coordinates (`groupId:artifactId`,
  `package.json` name, .NET project…) or, without descriptors, at least 80 % of their source paths.
  Different coordinates are never copies (they are different modules), and per-environment folders
  (`dev`, `test`, `uat`, `staging`, `prod`…) never are either.
- **`branches/` and `tags/`** next to `trunk/` are always left out, nested tags included.
- **Main line**: `trunk`; otherwise the highest version; on a tie, the most recent.
- **Not code**: documents, models (`.eap`, `.qea`, `.archimate`…), binaries and **runtimes committed
  to the repository** (a bundled JDK or application server). SQL scripts are analyzed (data model).
- **Binary-only modules**: a module with a `.war`/`.ear` and no source is read from the inside
  ([section 5.11](#511-modules-delivered-only-as-binaries)).

The decisions are written into the report (**Repository structure**). To analyze every copy anyway,
use `--layout all`. If a folder holds several projects with their own `trunk`, the survey says so,
in case you prefer to analyze them as separate systems. `survey --out DIR` also writes `survey.md`
and `survey.json`.

### 3.4 Very old code

- **Encodings**: files that are not valid UTF-8 are read as Windows-1252/ISO-8859-1 and accents are
  preserved. Affected files are listed in `files.csv` (`encoding`).
- **No Maven** (Ant, JARs in `WEB-INF/lib` or `lib/`): committed libraries are inventoried as
  `vendored-jar`, with their version when the name or embedded metadata gives it.
- **Java 1.4, EJB 2.x, Struts 1, Axis 1** have specific rules and are flagged as migration risks.
- **Descriptors with remote DTDs** (`web.xml` 2.3…) are read without downloading anything.

### 3.5 The portfolio manifest

`repos.csv`, UTF-8:

```csv
name,path,group
common-lib,common-lib,Shared
case-management,case-management,Case processing
notifications,notifications,Case processing
web-portal,portal,Channels
```

- `name`: system name in the reports.
- `path`: folder, absolute or relative to the CSV. Point it at a specific line of code (for example
  `app/trunk2`) to override the main-line choice.
- `group`: optional area, domain or team. Groups organize the map, the C4 models and the per-area
  views.
- Lines starting with `#` are ignored.

---

## 4. Run an analysis

### 4.1 Basic commands

```sh
traverse scan /src/app-c --out ~/reports/app-c                     # one repository
traverse scan /src/app-a /src/app-b --out ~/reports/portfolio      # several, as a portfolio
traverse scan --manifest repos.csv --out ~/reports/portfolio       # from a manifest (recommended)
```

```powershell
traverse.exe scan --manifest C:\src\repos.csv --out C:\reports\portfolio
```

**The output folder must be outside the analyzed repositories**: Traverse refuses to write inside
them. Each run overwrites the output folder; use a new one to compare runs.

### 4.2 Options

| Option | Default | Use |
|---|---|---|
| `--out DIR` | — (required) | Output folder |
| `--manifest FILE` | — | CSV `name,path,group` |
| `--lang en\|es` | `en` (or `TRAVERSE_LANG`) | Report language |
| `--name`, `--group` | folder name | Name and group when analyzing a single repository |
| `--config FILE` | — | `traverse.json`: exclusions, layers, rule packs, classification banner ([section 7](#7-configuration-custom-rules-and-rule-packs)) |
| `--rules FILE` | — | Extra rules or a rule pack (repeatable; the same `id` replaces a built-in rule) |
| `--no-default-rules` | off | Use only the `--rules` files |
| `--exclude GLOB` | — | Exclude paths, e.g. `--exclude "legacy/old-copy/**"` (repeatable) |
| `--keep-dir NAME` | — | Analyze a folder excluded by default (`bin`, `build`, `out`…) |
| `--include-tests` | off | Include test code in components, graphs and risks |
| `--layout auto\|all` | `auto` | `auto`: only the main line when there are copies. `all`: everything |
| `--svn-all` | off | Same as `--layout all` |
| `--key-file FILE` | `TRAVERSE_KEY_FILE` | Key from `traverse keygen`: enables **encrypted host names** ([section 8.1](#81-encrypted-host-names)) |
| `--include-hosts` | off | Require host capture: fail if no key is configured |
| `--vulndb PATH` | `TRAVERSE_VULNDB` | OSV dump downloaded beforehand (`all.zip` or folder; repeatable) |
| `--min-confidence low\|medium\|high` | `low` | Minimum confidence of the relationships **drawn** in diagrams. Tables and CSV keep them all |
| `--max-file-size MB` | 5 | Larger files are inventoried but not analyzed |
| `--max-nodes N` | 60 | Packages drawn in the layer diagram |
| `--classification TEXT` | — | Banner on every report, e.g. "Internal use" |
| `--workers N` | number of CPUs | Parallelism |

Always excluded (unless `--keep-dir`): `.git .svn CVS .idea .vscode .settings node_modules target
build dist out bin obj classes vendor packages __pycache__ .venv` and similar.

### 4.3 Other commands

```sh
traverse survey PATH...                         # structure: where the code is, copies, main line
traverse survey --out DIR PATH...               # also writes survey.md and survey.json
traverse rules list > rules.csv                 # rule catalog
traverse rules test --rules my-rules.json       # verify rule examples
traverse keygen --out team.key                  # key for encrypted host names
traverse reveal --key-file team.key REPORT_DIR  # decrypt host pseudonyms (CSV on stdout)
traverse vulndb info osv/2026-11-03             # check an OSV dump: ecosystems, advisories, data date
traverse version
```

### 4.4 Indicative performance

A synthetic 60 MB repository with 5,000 Java files takes about 26 s on 2 CPUs. Thirteen real
projects (about 6,000 files, 9 technologies) take about 8 s with every output. Repositories full of
branches and release copies take longer to walk, but those copies are skipped, not analyzed.

---

## 5. Read the results

> A sample report is in [examples/demo-report](../examples/demo-report/index.html): a fictional
> portfolio of 5 systems (Java/JBoss in SVN, Spring Boot in Git, .NET + COBOL, a shared library and
> a GitOps repository), generated with the **synthetic** vulnerability database in
> `examples/demo-vulndb`, which contains no real advisory.

### 5.1 Output folder

```
reports/portfolio/
├── index.html                ← portfolio report (start here)
├── portfolio.md              ← portfolio document, integration-centred (Markdown)
├── landscape.svg / .mmd / .puml
├── workspace.dsl             ← C4 model (Structurizr)
├── architecture.archimate.xml
├── findings.sarif            ← all systems
├── portfolio.json
├── systems.csv  links.csv  shared-tables.csv  library-versions.csv
├── external-dependencies.csv  technologies.csv
├── validation-sample.csv     ← 20 statements to measure reliability (5.10)
├── hosts.vault.json          ← encrypted host names (only with a key)
├── vulnerabilities.csv       ← only with --vulndb
├── areas/<group>/            ← per-area views, when the manifest has groups (5.9)
└── systems/<system>/
    ├── report.html           ← application report (with executive summary)
    ├── report.md             ← the same analysis in Markdown
    ├── arc42.md              ← pre-filled arc42 documentation
    ├── facts.json            ← every extracted fact (stable schema)
    ├── context.svg  architecture.svg  architecture.mmd  workspace.dsl  (deployment.svg)
    ├── findings.sarif  sbom.cdx.json
    ├── vulnerabilities.csv  vulnerabilities-unevaluated.csv     ← only with --vulndb
    └── files.csv  languages.csv  modules.csv  build-dependencies.csv  deployables.csv
        deployment.csv  components.csv  code-dependencies.csv  contracts.csv  findings.csv
        package-metrics.csv  cycles.csv  layer-violations.csv  diagnostics.csv
```

CSV files are UTF-8 with BOM (spreadsheets open them with the right accents) and formulas are
neutralized to prevent injection.

### 5.2 Portfolio report (`index.html`)

| Section | Answers |
|---|---|
| **System map** | Which systems exist and how they relate; colour = relationship type |
| **Systems** | Size, languages, modules, exposed and consumed interfaces, tables, runtime, deployment units and risks |
| **Relationships** | Each link with type, evidence count, confidence and keys (HTTP path, queue, table, artifact) |
| **Shared tables** | Tables used by several applications and how (read, write, define): a key sign of coupling through the database |
| **Shared libraries** | Libraries used by several systems and their **version divergence** |
| **Dependencies outside the portfolio** | Calls to services no analyzed system exposes: candidate external systems |
| **Technology matrix** | Technology × system |

**How relationships are found:**

| Type | Evidence matched |
|---|---|
| `library` | A Maven/Gradle/npm/NuGet dependency of one system = a module produced by another (or a committed JAR with the same name) |
| `http` | A path called by a client (literal URL, configured base URL, `@FeignClient`, `@RegisterRestClient`, `jaxws:client`) = a path exposed by another system (context root + application path + `@Path`/`@RequestMapping`). Versioned context roots (`/app-2.3/…`) match their unversioned form |
| `soap` | A generated client or configured endpoint = a `@WebService` or published endpoint of another system |
| `jms` / `kafka` | A destination produced by one = consumed by another (MDB, `@JmsListener`, `@KafkaListener`, `ejb-jar.xml`, Spring XML) |
| `shared-db` | The same table referenced by several systems (JPA `@Table`, SQL literals, `.sql` scripts, `EXEC SQL`) |
| `shared-datasource` | The same JNDI datasource name (low confidence) |
| `shared-dataset` | The same mainframe dataset (DSN): JCL, CICS files |
| `cics-transaction` / `program` | A CICS transaction or COBOL program invoked by one system and defined in another |
| `deploys` | A platform/GitOps repository deploys an image named after a module of the portfolio |

- **Shared tables**: matching only the table **name** (USERS exists everywhere) is **low**
  confidence; matching the **schema** (`CASES.CASE`) or the **datasource** raises it to **medium**.
  The `basis` column of `shared-tables.csv` says which.
- **Internal calls**: an HTTP call matching an endpoint of the same system is internal and is not
  linked to others. A match obtained only through path variables is low confidence.
- An HTTP path needs at least two literal segments to be linked, and the most specific match wins.

### 5.3 Application report (`systems/<system>/report.html`)

- **System context**: what it exposes (in), what it consumes (out), data and platform (C4 level 1).
- **Deployment units**: WAR/EAR/EJB with their *context root* and application paths.
- **Layered architecture**: packages per layer (Presentation, Business, Integration, Persistence,
  Model, Configuration, Cross-cutting) and their dependencies. **Red lines are forbidden
  dependencies**, e.g. Presentation → Persistence.
- **Layer violations** and **cycles**, with file and line.
- **Interfaces and integrations**: each contract with type, direction, role (API or screen), key,
  counterpart, component, transport, confidence and location.
- **Data access**: tables, access type and components.
- **Modules and build**: coordinates, version, packaging, Java/.NET level and dependencies
  (`dependencyManagement` entries are marked *managed* and do not count as use).
- **Technologies** and **Risks**, with severity, rule, file, lines and recommendation.
- **Components**: each class with its layer and the **reason**: `stereotype:@Stateless`,
  `extends:HttpServlet`, `package:dao`, `name:*Service`…
- **Package coupling**: Ca, Ce and instability I = Ce/(Ca+Ce).

Every table can be filtered and sorted. Reports work in light and dark mode and load nothing from the
Internet.

### 5.4 Deployment view

Infrastructure as code is read with a built-in reader, no external tools:

| Source | Extracted |
|---|---|
| **Dockerfile** | Base images per stage, ports, user, `ENV`/`ARG` names, `HEALTHCHECK` |
| **Docker Compose** (with overrides) | Services, image or build context (linked to the code module), ports, replicas, `depends_on`/`links`, references between services in environment variables (the service name is detected, the value never stored) |
| **Kubernetes** | Deployment, StatefulSet, DaemonSet, Job, CronJob, Pod, Service, Ingress (encrypted host), ConfigMap and Secret (keys only), HPA |
| **OpenShift** | DeploymentConfig, Route |
| **Helm** | Templates (`{{ }}` directives neutralized) |

Deployment risks: untagged or `latest` base image, unsupported base image (Java 8, Node < 18,
Python 2…), container running as root, `ADD` from URL, privileged container, no resource limits,
Secret with data in the repository.

### 5.5 Executive summary and Markdown

Every report starts with an **executive summary**: size and stack, deployment, integrations,
vulnerabilities, risks and coupling, in a few lines. The full analysis is also exported to
**`report.md`** per system, ready to version, review in a merge request or paste into a wiki.

### 5.6 Authentication, mTLS and third parties

The **Authentication and transport** section groups mechanisms by kind:

| Kind | Examples detected |
|---|---|
| **User login** | Container authentication (`web.xml`), Spring Security, a custom `AuthenticationProvider` (shown by class name), OAuth2/OIDC, JWT, SAML, CAS, Keycloak, Kerberos/SPNEGO, LDAP/AD, eIDAS, SSO headers from a gateway, JBoss `security-domain` |
| **Service to service** | WS-Security, API keys, Basic/Bearer credentials in clients |
| **E-signature** | XML-DSig, CMS/PKCS#7, PDF signatures, XAdES/CAdES/PAdES/JAdES (DSS), RFC 3161 timestamping, PKCS#11 tokens |
| **Transport (TLS/mTLS)** | `CLIENT-CERT`, Tomcat `clientAuth`, WildFly `verify-client`, Spring Boot `server.ssl.client-auth`, Liberty `clientAuthentication`, client keystores in code (`KeyManagerFactory`, `TLSClientParameters`…) |

If no login is visible in the code, the report says so (*possibly delegated to the infrastructure*).

**Inbound mTLS** is reported in one of three states:

| State | Evidence |
|---|---|
| **Required by the server** | The application server asks for client certificates (`CLIENT-CERT`, `clientAuth`, `verify-client`, `server.ssl.client-auth`…) |
| **At an intermediary (inferred)** | The code reads the client certificate, or data from it (subject, serial, verification result), from an HTTP header: `@RequestHeader`/`@HeaderParam`/`getHeader` with a certificate header, well-known names such as `SSL_CLIENT_CERT`, `X-SSL-Client-Cert`, `X-Client-Cert`, `X-Forwarded-Client-Cert` or `X-ARR-ClientCert`, or a configured certificate header. A proxy, API gateway or load balancer terminates mTLS and forwards the certificate |
| **Not visible** | Neither. An intermediary may still require client certificates without the application seeing them |

In the inferred case, **each endpoint** served by that code gets the transport *mTLS at intermediary
(inferred)*: the endpoints of the controller that reads the header, or every API endpoint of the
module when the header is read in a filter or interceptor. `contracts.csv` lists them with their
`transport` and `auth` columns. Who requires mTLS (a consumer, a regulation) and which consumers
connect is not in the code: `arc42.md` leaves a ✍️ item to record it.

- **Weak spots**: outbound calls over plain HTTP, SOAP without WS-Security, disabled TLS
  verification, obsolete TLS protocols.
- **Configuration integrations**: URLs in `.properties`, YAML, Spring XML, `standalone.xml` and
  `server.xml` (HTTP/SOAP, JDBC, LDAP, SMTP, FTP/SFTP, AMQP…) are outbound integrations, with
  transport, environments that declare them and encrypted host.
- **Counterparts are named from the code**: the client package, the configuration key, the JNDI name
  (`java:/Mail` → *mail server*), a **database schema of another system** or a **platform module**.
- **Keystores** are inspected without decrypting anything: with a private key, **high** risk; a
  truststore with only public certificates, **info**.
- **Third-party inventory**: every external system the application talks to, with direction,
  protocol, transport, authentication, environments and evidence.

> **Limit:** Traverse sees what is **configured** in the code and descriptors. It cannot tell that a
> load balancer **enforces** mTLS in production, nor discover inbound consumers that are not in the
> analyzed code.

### 5.7 Relationships: whom it calls and who calls it

Each application report has **Relationships with other applications**:

- **Calls**: one row per target application, with protocols, number of endpoints, transport,
  authentication, environments and evidence.
- **Called by**: the **analyzed** systems that call it. To know who consumes an interface, the
  consumer must be analyzed too, so make the portfolio as complete as possible.
- **Interface with several candidates**: when several analyzed systems implement the interface being
  called (typical of a callback every application implements), the relationship is medium confidence
  with the note "N systems implement the interface": run-time configuration decides the target.
- **No identified consumer**: exposed interfaces no analyzed system calls, split into API, screens
  and others.

### 5.8 Certificates

Public certificates in the repository (PEM/DER files, also without extension, and JKS truststore
entries) are read without decrypting anything:

- **trusted applications or servers**, **own identity** (certificates of private-key entries),
  **trusted CAs**;
- **expiry**: expired or expiring within 90 days produces a finding.

The subject and alias of a **natural person's certificate** are never stored. A person is recognized
from standard attributes only: given name, surname or pseudonym (X.520), a natural-person semantics
identifier (ETSI EN 319 412-1: `PAS`, `IDC`, `PNO`, `TAX`, `TIN` + country), an e-mail in a subject
that is not a host, or an ID number in the common name. A CA is never a person. Host names anywhere
in a subject or alias are replaced by `‹host›` and sealed.

### 5.9 Per-area views

When the manifest assigns groups, `areas/<group>/` holds a view per area (`index.html`,
`portfolio.md`, diagrams) with the systems of the area and, from other areas, only those related to
it. **Always run the analysis globally**: analyzing each area separately turns cross-area
relationships into external dependencies.

### 5.10 Measure reliability: `validation-sample.csv`

`validation-sample.csv` lists 20 statements chosen deterministically across five kinds:
relationships, outbound calls, authentication, high risks and structure. Each row has its evidence.
Ask someone who knows the applications to set the verdict: *correct*, *false positive*,
*incomplete* or *not verifiable*.

Precision = correct ÷ (correct + false positives). *Incomplete* rows show what is missing. The same
code always gives the same sample, so several reviewers can compare their verdicts. With 20 rows
the estimate is rough: review a second sample before quoting a figure.

### 5.11 Modules delivered only as binaries

A module with a `.war` or `.ear` and no source (a third-party product, for instance) is **read from
the inside, without extracting or running it**: `WEB-INF/web.xml` and other descriptors,
configuration in `WEB-INF/classes`, libraries in `WEB-INF/lib` and `lib/` (into the SBOM and the
vulnerability matching), and the artifact's real version, compared with the `pom.xml`
(`artifact-version-mismatch` diagnostic). Evidence points into the archive:
`app-2.4.war!/WEB-INF/web.xml`.

### 5.12 arc42 documentation

Each system has `arc42.md` with the 12 [arc42](https://arc42.org) sections, filled from the code;
✍️ marks what a person must complete.

| Section | Generated content |
|---|---|
| 1 Introduction | Size, languages, modules (goals ✍️) |
| 2 Constraints | Runtimes, servers, databases |
| **3 Context** | Context diagram and **integration catalog**: direction, type, counterpart, key, **mode** (synchronous, asynchronous, shared data, build or deploy time), component, `file:line`, confidence |
| 4 Solution strategy | Technologies by area |
| 5 Building blocks | Containers, module dependencies, components per layer, cycles, violations |
| 6 Runtime | Scenarios **derived statically**: from each entry point to the reachable outbound integrations |
| 7 Deployment | Deployment units, infrastructure as code, resources the server must provide |
| 8 Cross-cutting concepts | Security, persistence, logging, scheduling, configuration |
| 9 Decisions | ADRs found in the repository (decisions ✍️) |
| 10 Quality | Test ratio, cycles, violations, unclassified share (requirements ✍️) |
| 11 Risks | Prioritized risks with evidence |
| 12 Glossary | Candidate terms: entities, tables, transactions |

`portfolio.md` documents the whole: system catalog, map, **integration matrix**, **integration
catalog** with mode and confidence, shared data, libraries and versions, external dependencies.
Diagrams are Mermaid blocks that GitLab and GitHub render directly.

### 5.13 Low-confidence relationships

Candidates that need validation are drawn **dimmed** (thin, dashed, "low confidence") in SVG,
Mermaid, Structurizr (`low-confidence` tag) and PlantUML (`$tags="low"`). `--min-confidence medium`
hides them from diagrams; they stay in tables, CSV and the integration catalog.

### 5.14 Using the diagrams and models

| File | Tool (offline) | Use |
|---|---|---|
| `*.svg` | Browser, office suites, draw.io | Insert directly |
| `workspace.dsl` | Structurizr Lite / CLI | Editable C4 model |
| `landscape.puml` | `plantuml.jar` (C4 library included) | C4 diagrams |
| `*.mmd` | Mermaid CLI, GitLab, GitHub | Edit and version in a wiki |
| `architecture.archimate.xml` | Archi (*File → Import → Open Exchange XML Model*), Sparx EA, BiZZdesign | Enterprise architecture model |
| `findings.sarif` | GitLab, GitHub code scanning, SonarQube, DefectDojo, VS Code | Manage risks as findings |
| `sbom.cdx.json` | OWASP Dependency-Track, CycloneDX tools | Component inventory (with vulnerabilities when `--vulndb` is used) |

**ArchiMate 3.1 mapping:** system → *Application Component*; exposed interfaces → *Application
Interface*; tables and datasets → *Data Object* with *Access*; runtimes, servers and databases →
*System Software* with *Serving*; WAR/EAR → *Artifact* with *Realization*; calls → *Serving*;
messaging and shared datasets → *Flow*; shared tables → *Association*; manifest groups → *Grouping*.

> Do not upload these files to public web services (online Mermaid or PlantUML renderers…) without
> authorization: they describe internal architecture.

### 5.15 Known vulnerabilities (offline OSV)

Traverse matches library versions against **OSV** (osv.dev), which aggregates GitHub, PyPI, Go,
RustSec and other sources; most advisories carry their CVE. **Traverse connects to nothing**: you
download the data separately.

**1. Download** on a machine allowed to reach the Internet, only the ecosystems you need:
`https://osv-vulnerabilities.storage.googleapis.com/<Ecosystem>/all.zip` with `Maven`, `npm`,
`NuGet`, `PyPI`, `Packagist` or `Go`. A browser is enough, or:

```sh
sh scripts/fetch-osv.sh osv/2026-11-03 Maven npm
```

```powershell
.\scripts\fetch-osv.ps1 -Dest osv\2026-11-03 -Ecosystems Maven,npm
```

Keep each zip named after its ecosystem (`Maven-all.zip`…) or in one sub-folder per ecosystem, and
check it with `traverse vulndb info osv/2026-11-03`. An ecosystem shown as `partial` (present only
through advisories shared inside another ecosystem's zip) is **not evaluated** until you download
its own `all.zip`.

**2. Analyze** with `--vulndb osv/2026-11-03`.

**3. Results** appear in each application report (*Known vulnerabilities*), in `arc42.md` (11.1),
`portfolio.md`, `vulnerabilities.csv` (with a `validation` column), `findings.sarif` (`VULN/<id>`)
and `sbom.cdx.json`.

| Version source | Example | Confidence |
|---|---|---|
| Exact version in the build descriptor | `pom.xml`, `packages.config`, `requirements.txt` with `==`, `go.mod` | high |
| Version pinned by a committed lockfile, **transitives included** | `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `bun.lock`, `composer.lock`, `packages.lock.json`, `poetry.lock`, `uv.lock`, `Pipfile.lock`, `gradle.lockfile` | high |
| .NET central package management | `Directory.Packages.props`, `$(...)` properties included | high |
| Committed JAR with Maven metadata | `WEB-INF/lib/x.jar` with `pom.properties` | high |
| JAR identified only by file name | `commons-lang-2.6.jar` | medium, or low if ambiguous |
| .NET assembly reference | `<Reference Include="log4net, Version=1.2.10.0">` | low |

Nothing is silently assumed safe: every library that cannot be evaluated is listed in
`vulnerabilities-unevaluated.csv` with the reason: `external-bom` (version set by a parent POM or BOM
not in the repository), `range` (no lockfile), `unresolved` (`${...}` undefined), `missing`, or
`ecosystem-not-loaded`.

CVSS v3 base scores are computed with the official FIRST formula; known **malicious** packages
(`MAL-` ids) are critical. Data older than 30 days raises a `vulndb-stale` warning. Results are
**candidates**: OSV says the version is affected, not that the vulnerable code is used.

---

## 6. Validate the findings

Traverse works with evidence, not certainty. Before an official document:

1. **Prioritize**: high risks, relationships between systems, shared tables.
2. **Check the evidence**: every row has `file:line`; open it.
3. **Record the decision** in the `validation` column of `findings.csv`: `confirmed`, `probable`,
   `discarded` or `not-verifiable`.
4. **Ask** the application and infrastructure owners about what the code cannot show, such as who requires inbound mTLS
   ([Limitations](#9-limitations)).

Finding ids are **stable** (they do not depend on line numbers), so validations can be carried over to
a new run by `id`. To measure overall reliability, use `validation-sample.csv`
([section 5.10](#510-measure-reliability-validation-samplecsv)).

---

## 7. Configuration, custom rules and rule packs

### 7.1 `traverse.json`

```json
{
  "classification": "Internal use",
  "exclude": ["**/generated/**", "legacy/old-copy/**", "*.min.js"],
  "layers": {
    "overrides": {
      "com.example.cases.facade": "Business",
      "com.example.cases.ws": "Presentation"
    },
    "tokens": {
      "Business": ["claims", "underwriting"],
      "Integration": ["sap", "gateway"]
    }
  },
  "rules": ["rulepacks/my-organization.json"]
}
```

- `exclude`: paths not analyzed, relative to each repository.
- `layers.overrides`: force the layer of a package or class and everything below it (highest priority).
- `layers.tokens`: package words of your organization, in any language. Layers: `Presentation`,
  `Business`, `Integration`, `Persistence`, `Model`, `Configuration`, `Cross-cutting`.
- `rules`: extra rule files or packs, relative to the `traverse.json` file, loaded after the core
  catalog and before `--rules`.
- `classification`: banner printed on every report.

### 7.2 Custom rules

Rules are JSON with an RE2 regular expression and **mandatory examples**, verified automatically:

```json
{
  "version": "local-1",
  "rules": [{
    "id": "LOCAL-FWK",
    "title": "Legacy in-house framework",
    "category": "Framework", "technology": "In-house framework",
    "severity": "medium", "confidence": "high",
    "languages": ["Java"],
    "pattern": "\\borg\\.example\\.fwk\\.",
    "message": "Discontinued framework: plan its replacement.",
    "examples": {
      "match":   ["import org.example.fwk.base.BaseAction;"],
      "noMatch": ["// import org.example.fwk.base.BaseAction;"]
    }
  }]
}
```

```sh
traverse rules test --rules my-rules.json
traverse scan --manifest repos.csv --rules my-rules.json --out reports
```

- Rules run on code **without comments**: commented-out code produces no false positives.
- `pathPattern` (instead of `pattern`) matches file paths, e.g. `"\\.jks$"`.
- `exclude` drops a match whose text also matches it (e.g. a URL where a password was expected).
- `authKind` places an authentication rule in the security section: `user`, `service`,
  `signature` or `transport`.
- A rule with the same `id` as a built-in rule replaces it.

Field reference: [docs/rules.md](rules.md).

### 7.3 Rule packs

The core catalog only knows what is valid everywhere. Knowledge of one country, sector or
organization goes into **rule packs**: ordinary rule files in [`rulepacks/`](../rulepacks/), loaded
only when asked for, with `--rules` or the `rules` key of `traverse.json`.

| Pack | Scope |
|---|---|
| `es-public-sector.json` | Spanish public sector: @firma/AutoFirma, Cl@ve, FIRe, Valida, FNMT/CERES (`ES-*`) |

```sh
traverse scan --manifest repos.csv --rules rulepacks/es-public-sector.json --out reports
```

To write a pack for your country, sector or organization: prefix every id with its scope (`DE-`,
`ACME-`…) so it never replaces a core rule, name the `version` after the pack, give every rule
examples and run `traverse rules test --rules <pack>`. Contributions of new packs are welcome
([CONTRIBUTING.md](../CONTRIBUTING.md)).

---

## 8. Security and privacy

**Design guarantees**, verifiable in the source code:

- **Read-only**: analyzed files are only read. Output goes to `--out`, which must be outside the
  repositories.
- **No network**: there is no network code. XML DTDs/XSDs are never resolved (no XXE). `--vulndb`
  is read from disk; no API is queried, so your library list never leaves the machine.
- **Archives** (JAR, WAR, EAR, OSV zips) are read in memory, never extracted to disk or executed, with
  size and entry limits against malformed or malicious files.
- **Keystores and certificates**: Traverse only tells whether a keystore holds a private key and reads
  public certificates; nothing is decrypted.
- **No execution**: Maven, Gradle, git, svn, compilers or any process are never invoked.
- **No telemetry, no auto-update, no third-party code**: Go standard library only.

**Reports contain** relative paths, class and package names, HTTP paths, JNDI/queue/table names,
library coordinates and line numbers. **They never contain** code lines, property or environment
values, passwords or tokens (secret rules report the location only), credentials in URLs or
connection strings, full URLs, clear-text host names, or personal certificate data.

Details: [SECURITY.md](../SECURITY.md).

### 8.1 Encrypted host names

Host names (servers called by URL, image registries, Ingress/Route hosts, WSDL addresses,
certificate subjects) are useful for the external dependency catalog but reveal infrastructure:

1. **Without a key** no host is recorded.
2. **With a key** each host becomes a **stable pseudonym** (`host-70239857e852`, HMAC-SHA256): the
   same host always gets the same pseudonym, so you can see that two systems call the same server
   without knowing which one.
3. The real name is stored **AES-256-GCM encrypted** in `hosts.vault.json` (mode 600). Only key
   holders can decrypt it with `traverse reveal`.

```sh
traverse keygen --out team.key                                     # once; keep it out of repos and reports
traverse scan --manifest repos.csv --out reports --key-file team.key
traverse reveal --key-file team.key reports > hosts-clear.csv      # authorized people only
```

- The key can also be given with `TRAVERSE_KEY_FILE`, or as a passphrase with `TRAVERSE_PASSPHRASE`
  (at least 12 characters; PBKDF2-SHA256, 600,000 iterations). A key file is recommended.
- Use the **same key** for every run and every person, so pseudonyms can be compared.
- A lost key cannot be recovered: analyze again with a new one.

Reports are still **internal information**: store them in authorized locations, use
`--classification`, and do not upload them to external services without authorization.

---

## 9. Limitations

- **Static and heuristic.** No full classpath type resolution; reflection, AOP, dynamic configuration
  and run-time injection by name are invisible.
- **What is not in the repository does not exist for Traverse**: application-server configuration,
  load balancers, ESBs, per-environment URLs injected from outside, real database schemas. If you have
  exports (`standalone.xml`, `*-ds.xml`, `*-jms.xml`…), add them as one more repository.
- **HTTP relationships** need the path in the code or the repository's configuration.
- **Dynamic SQL**: tables in literals and scripts are found, not those built by complex concatenation.
- **Depth per language**: the deepest structural analysis is for Java, Kotlin and C#. In JS/TS,
  Python, PHP, Go and Ruby the component is the **module**, and route prefixes defined in another
  file are not resolved.
- **ASP.NET MVC without route attributes**: conventional `/{controller}/{action}`, low confidence.
- **CICS**: transfers to programs named at run time cannot be resolved (`dynamic-call` diagnostic).
- **Deployment**: Helm values and Kustomize overlays are not resolved; Terraform and Ansible are only
  detected.
- **Front-ends** (Angular, React, Vue): modules and imports; backend calls built with template strings
  are not linked.
- **Domain-organized packages** (`orders`, `users`) cannot be given a layer by name: use
  `layers.overrides`.
- **Vulnerabilities**: no transitive resolution without a lockfile (Maven and Gradle are never run);
  exotic version qualifiers may sort differently from the package manager.
- **Platforms**: tested on Linux and Windows 10/11. macOS binaries come from the same compiler but have
  not been tested on real projects.

---

## 10. Troubleshooting

| Problem | Solution |
|---|---|
| `output directory ... must be outside repository` | Use an output folder outside every analyzed repository |
| `not a directory` | Check the path in `repos.csv`; relative paths are resolved from the CSV folder |
| One repository fails and the others continue | Intended. Look for `FAILED` in the console |
| `traverse version` says `dev` | Built without `-ldflags "-X main.version=…"`; harmless |
| PowerShell: `The ampersand (&) character is not allowed` | Two commands were pasted on one line: run them separately or separate them with `;` |
| `permission denied` (Linux/macOS) | `chmod +x traverse` |
| Few components or dependencies | Is the code under `bin/`, `build/`, `out/` or `vendor/`? Use `--keep-dir`. Test code? `--include-tests`. Check **Repository structure**: is the main line the one you expected? If not, point at it |
| Inflated figures (the same code several times) | Copies without shared build coordinates. Run `traverse survey` and point at the right folder, or use `--exclude` |
| 0 components in a module | With only a `.war`/`.ear`, expected: it is read from the inside (5.11) |
| "Unclassified" layers | Add `layers.tokens` or `overrides` in `traverse.json` |
| Odd accents in a spreadsheet | Import the CSV as UTF-8 |
| Very large report | HTML tables show at most 2,500 rows; the CSV has everything. Lower `--max-nodes` if the diagram is dense |
| Skipped files | See `diagnostics.csv` (`too-big`, `read-error`, `xml-invalid`, `lockfile-partial`) |
| `no OSV advisories found in --vulndb` | The path has no OSV zips or JSON: check it with `traverse vulndb info PATH` |
| Many "unevaluated" libraries | See the reasons in `vulnerabilities-unevaluated.csv` (5.15) |
| A relationship you expected is missing | Look for the exposed and the consumed key in `contracts.csv` of both systems. If one is missing, the information is not in the code |

---

## 11. Appendix: locked-down machines

If your machine blocks executables that come from outside:

1. **Build from source on the machine itself** ([section 2.2](#22-build-from-source)). You only need
   the Go archive and the Traverse source; source code usually passes controls that block binaries.
2. **Build in your organization's CI** ([section 2.4](#24-build-in-gitlab-ci-or-github-actions)) and
   ask for the executable to be allowed by hash or signature. A request can read:

   > Request to allow the static analysis tool Traverse (open source, Apache-2.0).
   > File `traverse.exe`, SHA-256 `<hash>`. It reads source files and writes HTML/CSV reports to a
   > local folder. No installation or privileges, no network connections, runs no processes and
   > does not modify the analyzed files (see `SECURITY.md`). Built only with the Go standard library;
   > the source is available for audit.

3. **Run it on another authorized machine** (a Linux server, WSL, a virtual desktop, a CI runner) on
   a copy of the code, and copy the report folder back: reports are static HTML that opens offline.

**Never try to bypass security controls** (renaming, packing or altering the executable so it is not
detected): ask for authorization.
