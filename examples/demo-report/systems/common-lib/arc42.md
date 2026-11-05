# common-lib — Architecture documentation (arc42)

| Item | Value |
|---|---|
| System | common-lib |
| Group | — |
| Version control | none |
| Generated | 2025-11-05T00:00:00Z · Traverse dev · rules 2.0.0 |

> Generated from source code using the [arc42](https://arc42.org) template. Sections marked ✍️ need business knowledge or decisions that are not in the code. Everything extracted is static evidence: validate it before treating it as final.

## 1. Introduction and goals

> ✍️ **To complete:** purpose, key requirements, quality goals and stakeholders.

Size: 5 lines of code · languages: Java (5) · modules: 1 · components: 2.

**Analysis scope (repository structure)**

- Analyzed: `.` (maven:org.example:common-dto 2.3.0)

## 2. Constraints

> ✍️ **To complete:** organisational, legal and business constraints.

### 2.1 Detected technical constraints

| Kind | Constraint | Evidence |
|---|---|---|
| Runtime | java 1.8 | pom.xml |

## 3. Context and scope

### 3.1 Business context

> ✍️ **To complete:** users, business areas and processes it serves.

### 3.2 Technical context

```mermaid
flowchart LR
  n_SYS["common-lib"]
  n_case_management["case-management"]
  n_notifications["notifications"]
  n_case_management -.->|"library"| n_SYS
  n_notifications -.->|"library"| n_SYS
```

### 3.3 Integration catalog

Each row is an integration with its evidence in the code. **Direction**: inbound (others use this system), outbound (this system uses others) or shared (coupling through data). **Mode**: synchronous, asynchronous, shared data, build time or deployment.

| # | Direction | Kind | Counterpart | Key | Mode | Component | Evidence | Confidence |
|---|---|---|---|---|---|---|---|---|
| 1 | inbound | library | case-management | maven:org.example:common-dto | build time (library) | — | — | high |
| 2 | inbound | library | notifications | maven:org.example:common-dto | build time (library) | — | — | high |

**Summary by counterpart**

| Counterpart | Integrations |
|---|---|
| case-management | library ×1 |
| notifications | library ×1 |

> ✍️ **To complete:** service levels, volumes, security (authentication, encryption) and owner of each integration.

## 4. Solution strategy

> ✍️ **To complete:** fundamental decisions and their rationale.

Detected technologies:

_Nothing detected._

## 5. Building block view

### 5.1 Level 1: containers (build modules and deployment units)

| Module | Build | Packaging | Runtime | Deploys | Context root | Descriptor |
|---|---|---|---|---|---|---|
| common-dto | maven | jar | java 1.8 | — | — | pom.xml |

### 5.2 Level 2: components by layer

| Layer | Components | Examples |
|---|---|---|
| Model | 1 | CaseDTO |
| Cross-cutting | 1 | DateUtils |

```mermaid
flowchart LR
  subgraph L_Model["Model"]
    p_org_example_common_dto["org.example.common.dto (1)"]
  end
  subgraph L_Cross_cutting["Cross-cutting"]
    p_org_example_common_util["org.example.common.util (1)"]
  end
```

Dependency cycles: 0 · Layer violations: 0. Details in `cycles.csv` and `layer-violations.csv`.

> ✍️ **To complete:** responsibility of each block and its relevant internal interfaces.

## 6. Runtime view

Scenarios **derived statically**: from each entry point, code dependencies are followed to the reachable outbound integrations. They do not reflect actual execution order or conditions.

_Nothing detected._


> ✍️ **To complete:** main business scenarios, error flows and batch processes with their schedule.

## 7. Deployment view

### 7.1 Deployment units

_Nothing detected._

### 7.2 Infrastructure as code

_Nothing detected._

> ✍️ **To complete:** environments, physical or virtual nodes, networks, load balancers and per-environment configuration.

## 8. Cross-cutting concepts

_Nothing detected._

> ✍️ **To complete:** security model, error handling, transactions, auditing, internationalisation.

> ✍️ **To complete:** transport requirements imposed by third parties (for example, a consumer that requires mTLS) and the intermediary that enforces them (proxy, API gateway, load balancer).

## 9. Architecture decisions

> ✍️ **To complete:** relevant decisions, alternatives and consequences (ADR format recommended).

## 10. Quality requirements

> ✍️ **To complete:** quality tree and measurable scenarios.

Static indicators (indicative):

| Indicator | Value |
|---|---|
| Test code / main code | 0.00 (0 / 5 LOC) |
| Package or module cycles | 0 |
| Layer violations | 0 |
| Components without a recognisable layer | 0 % |

## 11. Risks and technical debt

_Nothing detected._

### 11.1 Known vulnerabilities

Source: OSV.dev (demo-vulndb) · data as of 2026-10-28T00:00:00Z · library versions checked: 0

No known vulnerability in the evaluated versions (this does not mean the libraries are safe: see the unevaluated list and the data date).

_Offline match of declared and committed library versions against the OSV database given with --vulndb. Confidence: high = exact coordinates; medium/low = identified by JAR file name or assembly version. Transitive dependencies are evaluated only when the repository has a lockfile (no build is executed)._

> ✍️ **To complete:** business impact, likelihood and mitigation plan for each risk.

## 12. Glossary

Candidate terms (domain entities, tables, transactions):

_Nothing detected._

