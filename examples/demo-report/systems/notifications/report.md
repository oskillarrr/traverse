# notifications — Architecture analysis

_git · revision 4f2a9c1e7b3d · no copies detected: everything is analyzed · generated 2025-11-05T00:00:00Z · traverse dev_

## Executive summary

- **Size and stack.** 86 lines of code across 9 files; main language Java (63%). Build modules: 1 · components: 5.
- **Deployment.** service /notifications.
- **Integrations.** Exposes 4 interfaces (http ×4) and consumes 4 external ones (database ×2, http ×1, jms ×1); 1 data tables; 0 JNDI resources.
- **Relationships.** Calls 2 applications (database, sms); called by 1 analyzed systems; 3 APIs without identified consumer.
- **Interfaces.** API for other systems: 4 · screens: 0 · unclassified: 0.
- **Risks.** Findings: 1 high · 0 medium · 1 low.
- **Coupling.** 0 package cycles (largest, 0 packages) and 0 layer violations.
- **Security.** user login not visible in the code · mTLS detected · 1 interface with mTLS at an intermediary (inferred) · weak spots: 1 outbound call over plaintext HTTP.
- **Third parties.** 5 connected applications (2 outside the portfolio), 7 integrations with them.

## Repository structure

- Analyzed: `.` (maven:org.example:notifications 3.0.1)

## Metrics

| Metric | Value |
|---|---|
| Lines of code | 86 |
| Files | 9 |
| Build modules | 1 |
| Components | 5 |
| Exposed interfaces | 4 |
|   · API for other systems / screens | 4 / 0 |
| Outbound calls | 4 |
| Data tables | 1 |
| JNDI resources | 0 |
| Package cycles | 0 |
| Layer violations | 0 |
| Findings (high/medium/low) | 1 / 0 / 1 |
| Vulnerabilities (critical/high) | 0 / 0 |

## Languages

| Language | Category | Files | Code |
|---|---|---|---|
| Java | code | 5 | 55 |
| YAML | config | 1 | 13 |
| Dockerfile | config | 1 | 10 |
| Maven POM | build | 1 | 5 |
| Properties | config | 1 | 3 |

## Technologies

| Area | Technology | Severity | Files |
|---|---|---|---|
| Database | Oracle Database | — | 2 |
| Database | Oracle SQL | low | 1 |
| Delivery | Docker | — | 1 |
| Framework | Spring Boot | — | 1 |
| Framework | Spring Framework | — | 2 |
| Integration | HTTP client | — | 1 |
| Integration | JMS | — | 1 |
| Persistence | Spring JDBC | — | 1 |
| Transport Security | mTLS at intermediary (inferred) | — | 1 |
| Web | Spring MVC | — | 3 |

## Integrations

| # | Direction | Type | Use | Counterpart | Key | Mode | Transport | Auth | Component | Evidence | Confidence |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | inbound | http | API | (no identified consumer) | GET /notifications/api/messages/by-id/{id} | synchronous | — | — | MessageController | src/main/java/org/example/notify/api/MessageController.java:12 | high |
| 2 | inbound | http | API | (no identified consumer) | GET /notifications/api/messages/{id} | synchronous | — | — | MessageController | src/main/java/org/example/notify/api/MessageController.java:12 | high |
| 3 | inbound | http | API | (no identified consumer) | POST /notifications/api/receipts/delivery | synchronous | mTLS at intermediary (inferred) | client-cert (header) | DeliveryReceiptController | src/main/java/org/example/notify/api/DeliveryReceiptController.java:13 | high |
| 4 | inbound | http | API | case-management | POST /notifications/api/messages/send | synchronous | — | — | MessageController | src/main/java/org/example/notify/api/MessageController.java:10 | medium |
| 5 | inbound | jms | — | case-management | java:/jms/queue/CaseResolved | asynchronous (messaging) | — | — | MessageService | src/main/java/org/example/notify/service/MessageService.java:9 | high |
| 6 | inbound | deploys | — | platform-gitops | openshift DeploymentConfig notifications ← tramitacion/notifications:3.0.1 | deployment | — | — | — | — | medium |
| 7 | outbound | library | — | common-lib | maven:org.example:common-dto | build time (library) | — | — | notifications | pom.xml:5 | high |
| 8 | outbound | database | — | database · host-bb8f9d8bfaea | oracle | through shared data | — | — | — | docker-compose.yml:7 | medium |
| 9 | outbound | database | — | database · host-240cd640a7e5 | oracle | through shared data | — | — | — | src/main/resources/application.properties:2 | medium |
| 10 | outbound | http | — | sms · host-fc2d5daa3ef0 | ANY /api/v1/sms | synchronous | plaintext | — | SmsGatewayClient | src/main/java/org/example/notify/sms/SmsGatewayClient.java:7 | medium |
| 11 | shared | shared-db | — | case-management | EXPEDIENTE | through shared data | — | — | CaseReadRepository | src/main/java/org/example/notify/repository/CaseReadRepository.java:7 | medium |
| 12 | shared | shared-db | — | legacy-portal | EXPEDIENTE | through shared data | — | — | — | — | medium |

## Authentication and transport security

- **Inbound mTLS:** inferred: 1 interface receives the client certificate in an HTTP header, a sign that an intermediary (proxy, API gateway, load balancer) terminates mTLS. Who requires it and which consumers use it is not in the code.

- **User login:** not visible in the code (possibly delegated to the infrastructure: gateway, SSO or server).

| Kind | Mechanism | Files | Evidence |
|---|---|---|---|
| Transport (TLS/mTLS) | mTLS at intermediary (inferred) | 1 | src/main/java/org/example/notify/api/DeliveryReceiptController.java:13 |

- **mTLS / client certificate** detected.

**Weak spots**

- 1 outbound call over plaintext HTTP

## Relationships with other applications

### What it calls (2)

| Application | Scope | Protocols | Endpoints | Transport | Auth | Environments | Evidence |
|---|---|---|---|---|---|---|---|
| database | external | database | 1 | — | — | — | docker-compose.yml:7 |
| sms | external | http | 1 | plaintext | — | — | src/main/java/org/example/notify/sms/SmsGatewayClient.java:7 |

### Who calls it

| Application | Protocols | Interfaces | Evidence |
|---|---|---|---|
| case-management | http, jms | 2 | src/main/java/org/example/notify/api/MessageController.java:10 |

- **Shares data with:** case-management, legacy-portal
- **No identified consumer:** 3 API · 0 screens · 0 other. Who consumes an interface is only known when the consumer is analyzed too.

## Connected applications inventory (third parties)

| Counterpart | Scope | Direction | Protocol | Transport | Auth | Endpoint | Environments | Hosts | Evidence | Confidence |
|---|---|---|---|---|---|---|---|---|---|---|
| case-management | portfolio | inbound | http | — | — | POST /notifications/api/messages/send | — | — | src/main/java/org/example/notify/api/MessageController.java:10 | medium |
| case-management | portfolio | inbound | jms | — | — | java:/jms/queue/CaseResolved | — | — | src/main/java/org/example/notify/service/MessageService.java:9 | high |
| case-management | portfolio | shared | shared-db | — | — | EXPEDIENTE | — | — | src/main/java/org/example/notify/repository/CaseReadRepository.java:7 | medium |
| database | external | outbound | database | — | — | oracle | — | 2 | docker-compose.yml:7 | medium |
| legacy-portal | portfolio | shared | shared-db | — | — | EXPEDIENTE | — | — | — | medium |
| platform-gitops | portfolio | inbound | deploys | — | — | openshift DeploymentConfig notifications ← tramitacion/notifications:3.0.1 | — | — | — | medium |
| sms | external | outbound | http | plaintext | — | ANY /api/v1/sms | — | 1 | src/main/java/org/example/notify/sms/SmsGatewayClient.java:7 | medium |

## Data access

| Table | Access | Components | Files |
|---|---|---|---|
| CASES.EXPEDIENTE | read | CaseReadRepository | 1 |

## Deployment

| Module | Kind | Context root | Source |
|---|---|---|---|
| example:notifications | service | /notifications | application properties |

| Node | Platform | Kind | Images | Evidence |
|---|---|---|---|---|
| broker | compose | service | apache/activemq-artemis:latest | docker-compose.yml:13 |
| notifications | compose | service | — | docker-compose.yml:3 |
| oracle | compose | service | gvenzl/oracle-xe:21-slim | docker-compose.yml:11 |
| root | docker | image | — | Dockerfile:1 |

## Known vulnerabilities

Source: OSV.dev (demo-vulndb) · data as of 2026-10-28T00:00:00Z · library versions checked: 1

No known vulnerability in the evaluated versions (this does not mean the libraries are safe: see the unevaluated list and the data date).

**Libraries whose version could not be evaluated (1)**

| Library | Version | Reason |
|---|---|---|
| maven:org.springframework.boot:spring-boot-starter-web | — | managed by a parent POM or BOM outside the repository |

## Risks and findings

| Severity | Finding | Category | Files |
|---|---|---|---|
| High | Possible hard-coded credential in configuration | Security | 1 |
| Low | Oracle-specific SQL | Database | 1 |

## Coupling

---
_Generated by static analysis; the code is never executed. Companion open formats: arc42.md, CycloneDX, SARIF, CSV and diagrams (C4, ArchiMate, Mermaid). ._
