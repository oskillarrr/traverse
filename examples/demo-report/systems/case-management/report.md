# case-management — Architecture analysis

_svn · main line trunk (1.4.2), 2 copies left out [trunk] · generated 2025-11-05T00:00:00Z · traverse dev_

## Executive summary

- **Structure.** analyzed `trunk` (1.4.2); 2 copies of the same project left out.
- **Size and stack.** 116 lines of code across 18 files; main language Java (71%). Build modules: 4 · components: 7.
- **Deployment.** ear, ejb, war /cases.
- **Integrations.** Exposes 3 interfaces (http ×3) and consumes 1 external ones (http ×1); 4 data tables; 3 JNDI resources.
- **Relationships.** Calls 1 applications (notifications); called by 1 analyzed systems; 1 APIs without identified consumer.
- **Interfaces.** API for other systems: 2 · screens: 0 · unclassified: 1.
- **Vulnerabilities.** 1 critical and 1 high of 2 known. Most severe: maven:commons-collections:commons-collections 3.2.1 (TRAVERSEDEMO-2026-0001, fixed in 3.2.2).
- **Risks.** Findings: 3 high · 8 medium · 2 low. Technology at risk: DB network, HttpClient 3, JBoss/WildFly, Java EE (javax), Log4j 1.x (!), java 1.6 (!).
- **Runtime.** Outdated runtime level: java 1.6.
- **Coupling.** 0 package cycles (largest, 0 packages) and 1 layer violations.
- **Security.** Container (JAAS), Container auth (CLIENT-CERT), AdES, Timestamping, X.509 client auth · mTLS detected.
- **Third parties.** 3 connected applications (0 outside the portfolio), 8 integrations with them.

## Repository structure

- Analyzed: `trunk` (maven:org.example.cases:case-parent 1.4.2) — main line (trunk)
- Copies of the same project left out: 2 (0 identical to the main line)
- `trunk/case-web/src/main/webapp/WEB-INF/lib`: binaries (1 files)
- `trunk/db`: SQL scripts (data model, analyzed) (1 files)

| Copy left out | Version | Relation | Overlap with main line | Files |
|---|---|---|---|---|
| branches/release-1.0 | — | copy | 0% | 1 |
| tags/v1.0 | — | copy | 0% | 1 |

## Metrics

| Metric | Value |
|---|---|
| Lines of code | 116 |
| Files | 18 |
| Build modules | 4 |
| Components | 7 |
| Exposed interfaces | 3 |
|   · API for other systems / screens | 2 / 0 |
| Outbound calls | 1 |
| Data tables | 4 |
| JNDI resources | 3 |
| Package cycles | 0 |
| Layer violations | 1 |
| Findings (high/medium/low) | 3 / 8 / 2 |
| Vulnerabilities (critical/high) | 1 / 1 |

## Languages

| Language | Category | Files | Code |
|---|---|---|---|
| Java | code | 7 | 83 |
| XML | config | 5 | 19 |
| Maven POM | build | 4 | 10 |
| SQL | code | 1 | 4 |
| Binary library | library | 1 | 0 |

## Technologies

| Area | Technology | Severity | Files |
|---|---|---|---|
| Authentication | AdES | — | 1 |
| Authentication | Container (JAAS) | — | 1 |
| Authentication | Container auth (CLIENT-CERT) | — | 1 |
| Authentication | Timestamping | — | 1 |
| Authentication | X.509 client auth | — | 2 |
| Business | EJB | — | 2 |
| Database | Oracle Database | — | 3 |
| Database | Oracle SQL | low | 1 |
| Database | PL/SQL | — | 1 |
| Delivery | Vendored libraries | low | 1 |
| Integration | DB network | medium | 1 |
| Integration | HTTP client | — | 1 |
| Integration | HttpClient 3 | medium | 1 |
| Integration | JMS | — | 1 |
| Integration | JNDI | — | 2 |
| Logging | Log4j 1.x | high | 1 |
| Migration | Java EE (javax) | medium | 4 |
| Migration | java 1.6 | high | 1 |
| Persistence | Hibernate | — | 1 |
| Persistence | JDBC | — | 1 |
| Persistence | JPA | — | 1 |
| Platform | JBoss/WildFly | medium | 3 |
| Transport Security | Custom TLS | — | 1 |
| Transport Security | TLS required | — | 1 |
| Transport Security | mTLS client | — | 1 |
| Web | JAX-RS | — | 1 |
| Web | Servlet | — | 1 |

## Integrations

| # | Direction | Type | Use | Counterpart | Key | Mode | Transport | Auth | Component | Evidence | Confidence |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | inbound | http | — | (no identified consumer) | ANY /cases/export/legacy | synchronous | — | — | LegacyExportServlet | trunk/case-web/src/main/webapp/WEB-INF/web.xml:7 | high |
| 2 | inbound | http | API | (no identified consumer) | POST /cases/api/expedientes/{id}/resolver | synchronous | — | — | ExpedienteResource | trunk/case-web/src/main/java/org/example/cases/web/ExpedienteResource.java:23 | high |
| 3 | inbound | http | API | legacy-portal | GET /cases/api/expedientes/{id} | synchronous | — | — | ExpedienteResource | trunk/case-web/src/main/java/org/example/cases/web/ExpedienteResource.java:19 | medium |
| 4 | inbound | deploys | — | platform-gitops | kubernetes CronJob nightly-reconciliation ← tramitacion/case-ejb:1.4.2 | deployment | — | — | — | — | medium |
| 5 | inbound | deploys | — | platform-gitops | kubernetes Deployment case-management ← tramitacion/case-web:1.4.2 | deployment | — | — | — | — | medium |
| 6 | outbound | library | — | common-lib | maven:org.example:common-dto | build time (library) | — | — | case-parent | trunk/pom.xml:4 | high |
| 7 | outbound | http | — | notifications | POST /notifications/api/messages/send | synchronous | mTLS | client-cert | NotificacionesClient | trunk/case-ejb/src/main/java/org/example/cases/integracion/NotificacionesClient.java:8 | medium |
| 8 | outbound | jms | — | notifications | java:/jms/queue/CaseResolved | asynchronous (messaging) | — | — | GestorExpedientes | trunk/case-ejb/src/main/java/org/example/cases/servicio/GestorExpedientes.java:12 | high |
| 9 | outbound | jndi | — | server resource | java:comp/env/jdbc/CasesDS | synchronous | — | — | ExpedienteDao | trunk/case-ejb/src/main/java/org/example/cases/dao/ExpedienteDao.java:10 | medium |
| 10 | outbound | jndi | — | server resource | java:jboss/datasources/CasesDS | synchronous | — | — | — | trunk/case-ejb/src/main/resources/META-INF/persistence.xml:1 | high |
| 11 | outbound | jndi | — | server resource | jdbc/CasesDS | synchronous | — | — | — | trunk/case-web/src/main/webapp/WEB-INF/web.xml:8 | high |
| 12 | shared | shared-db | — | legacy-portal | EXPEDIENTE | through shared data | — | — | Expediente | trunk/case-ejb/src/main/java/org/example/cases/modelo/Expediente.java:5 | medium |
| 13 | shared | shared-db | — | legacy-portal | TRAMITE | through shared data | — | — | — | trunk/db/schema.sql:3 | medium |
| 14 | shared | shared-db | — | notifications | EXPEDIENTE | through shared data | — | — | — | trunk/db/schema.sql:2 | medium |

## Authentication and transport security

- **Inbound mTLS:** required by the application server itself (see transport mechanisms).

| Kind | Mechanism | Files | Evidence |
|---|---|---|---|
| User login | Container (JAAS) | 1 | trunk/case-web/src/main/webapp/WEB-INF/jboss-web.xml:1 |
| User login | Container auth (CLIENT-CERT) | 1 | trunk/case-web/src/main/webapp/WEB-INF/web.xml:13 |
| E-signature | AdES | 3 | trunk/case-ejb/src/main/java/org/example/cases/signing/CaseResolutionSigner.java:2 |
| E-signature | Timestamping | 3 | trunk/case-ejb/src/main/java/org/example/cases/signing/CaseResolutionSigner.java:3 |
| Transport (TLS/mTLS) | Custom TLS | 3 | trunk/case-ejb/src/main/java/org/example/cases/integracion/NotificacionesClient.java:4 |
| Transport (TLS/mTLS) | TLS required | 1 | trunk/case-web/src/main/webapp/WEB-INF/web.xml:11 |
| Transport (TLS/mTLS) | X.509 client auth | 2 | trunk/case-web/src/main/webapp/WEB-INF/web.xml:13 |
| Transport (TLS/mTLS) | mTLS client | 3 | trunk/case-ejb/src/main/java/org/example/cases/integracion/NotificacionesClient.java:3 |

- **mTLS / client certificate** detected.

## Relationships with other applications

### What it calls (1)

| Application | Scope | Protocols | Endpoints | Transport | Auth | Environments | Evidence |
|---|---|---|---|---|---|---|---|
| notifications | portfolio | http, jms | 2 | mTLS | client-cert | — | trunk/case-ejb/src/main/java/org/example/cases/integracion/NotificacionesClient.java:8 |

### Who calls it

| Application | Protocols | Interfaces | Evidence |
|---|---|---|---|
| legacy-portal | http | 1 | trunk/case-web/src/main/java/org/example/cases/web/ExpedienteResource.java:19 |

- **Shares data with:** legacy-portal, notifications
- **No identified consumer:** 1 API · 0 screens · 1 other. Who consumes an interface is only known when the consumer is analyzed too.

## Connected applications inventory (third parties)

| Counterpart | Scope | Direction | Protocol | Transport | Auth | Endpoint | Environments | Hosts | Evidence | Confidence |
|---|---|---|---|---|---|---|---|---|---|---|
| legacy-portal | portfolio | inbound | http | — | — | GET /cases/api/expedientes/{id} | — | — | trunk/case-web/src/main/java/org/example/cases/web/ExpedienteResource.java:19 | medium |
| legacy-portal | portfolio | shared | shared-db | — | — | EXPEDIENTE | — | — | trunk/case-ejb/src/main/java/org/example/cases/modelo/Expediente.java:5 | medium |
| legacy-portal | portfolio | shared | shared-db | — | — | TRAMITE | — | — | trunk/db/schema.sql:3 | medium |
| notifications | portfolio | outbound | http | mTLS | client-cert | POST /notifications/api/messages/send | — | — | trunk/case-ejb/src/main/java/org/example/cases/integracion/NotificacionesClient.java:8 | medium |
| notifications | portfolio | outbound | jms | — | — | java:/jms/queue/CaseResolved | — | — | trunk/case-ejb/src/main/java/org/example/cases/servicio/GestorExpedientes.java:12 | high |
| notifications | portfolio | shared | shared-db | — | — | EXPEDIENTE | — | — | trunk/db/schema.sql:2 | medium |
| platform-gitops | portfolio | inbound | deploys | — | — | kubernetes CronJob nightly-reconciliation ← tramitacion/case-ejb:1.4.2 | — | — | — | medium |
| platform-gitops | portfolio | inbound | deploys | — | — | kubernetes Deployment case-management ← tramitacion/case-web:1.4.2 | — | — | — | medium |

## Data access

| Table | Access | Components | Files |
|---|---|---|---|
| CASES.EXPEDIENTE | read | Expediente | 2 |
| CASES.TRAMITE | read | — | 1 |
| EXPEDIENTE | read/write | ExpedienteDao | 1 |
| TRAMITE | read | ExpedienteDao | 1 |

## Deployment

| Module | Kind | Context root | Source |
|---|---|---|---|
| cases:case-ear | ear | — | — |
| cases:case-ejb | ejb | — | — |
| cases:case-web | war | /cases | application.xml |

## Known vulnerabilities

Source: OSV.dev (demo-vulndb) · data as of 2026-10-28T00:00:00Z · library versions checked: 3

| Severity | ID | Library | Fixed in | Confidence | Summary |
|---|---|---|---|---|---|
| Critical 9.8 | [TRAVERSEDEMO-2026-0001](https://osv.dev/vulnerability/TRAVERSEDEMO-2026-0001) | maven:commons-collections:commons-collections 3.2.1 | 3.2.2 | high | SYNTHETIC DEMO ADVISORY - not real vulnerability data. Remote code execution through a deserialization gadget |
| High 8.1 | [TRAVERSEDEMO-2026-0002](https://osv.dev/vulnerability/TRAVERSEDEMO-2026-0002) | maven:log4j:log4j 1.2.17 | — | high | SYNTHETIC DEMO ADVISORY - not real vulnerability data. Unsafe network appender in an end-of-life logging library |

## Risks and findings

| Severity | Finding | Category | Files |
|---|---|---|---|
| High | Log4j 1.x (end of life, CVEs) | Logging | 1 |
| High | Outdated runtime level: java 1.6 | Migration | 1 |
| High | Possible hard-coded credential in code | Security | 1 |
| Medium | javax.* Java EE namespace (pre Jakarta EE 9) | Migration | 4 |
| Medium | Commons HttpClient 3.x (end of life) | Integration | 1 |
| Medium | JBoss / WildFly proprietary API | Platform | 1 |
| Medium | Artifact repository over plain HTTP | Security | 1 |
| Medium | Database-initiated network/file access | Integration | 1 |
| Low | Libraries committed to the repository | Delivery | 1 |
| Low | Oracle-specific SQL | Database | 1 |

## Coupling

**Layer violations**

| From → to | Cases |
|---|---|
| Presentation → Persistence | 1 |

---
_Generated by static analysis; the code is never executed. Companion open formats: arc42.md, CycloneDX, SARIF, CSV and diagrams (C4, ArchiMate, Mermaid). ._
