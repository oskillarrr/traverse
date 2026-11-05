# Changelog

## 0.9.4 – 2026-11-05

mTLS entrante terminado en un intermediario, a partir de una sugerencia de revisión: cuando un
proxy o API gateway termina el mTLS, la aplicación solo recibe una petición HTTP con el certificado
en una cabecera.

- **Tres reglas nuevas, `TLS-UPSTREAM-*`**: certificado de cliente leído de una cabecera
  (`@RequestHeader`, `@HeaderParam`, `getHeader`, `Request.Headers`, `req.header`, `META`, y nombres
  habituales como `SSL_CLIENT_CERT`, `X-Client-Cert`, `X-Forwarded-Client-Cert` o `X-ARR-ClientCert`),
  datos del certificado en cabeceras (sujeto, número de serie, verificación) y cabecera de certificado
  configurada.
- **Por endpoint**: las interfaces del controlador que lee la cabecera, o todas las API del módulo si
  se lee en un filtro o interceptor, salen con transporte *mTLS en intermediario (inferido)* y
  autenticación `client-cert (header)`.
- **Estado del mTLS entrante** en el informe, `report.md`, el resumen ejecutivo y `arc42.md`: lo exige
  el servidor, inferido en un intermediario, o no visible en el código. `arc42.md` añade un apartado ✍️
  para los requisitos de transporte impuestos por terceros.
- **`contracts.csv`** añade al final las columnas `transport`, `auth`, `role` y `counterpart`.
- La demo incluye un endpoint que recibe el certificado del socio por cabecera.

## 0.9.3 – 2026-11-05

Muestra de validación revisada contra el código real: de 20 afirmaciones, 18 correctas, 1
incompleta, 1 no verificable desde el código y ningún falso positivo.

- **Nueva regla `SEC-SECRET-URL`**: usuario y contraseña dentro de una cadena de conexión
  (`esquema://usuario:clave@host`, `jdbc:oracle:thin:usuario/clave@…`). El valor nunca se copia al
  informe.
- **Contraseñas de keystore en código** con nombres como `JKS_KEY`, `KEYSTORE_PASS` o
  `TRUSTSTORE_PASSWORD`.
- **Redacción de la muestra**: un módulo del servidor o un recurso JNDI se "usa", no se "llama".
- **Documentación para un repositorio público**:
  - nueva guía de referencia en inglés, `docs/manual.md`, genérica y con cada comando para
    Linux/macOS y para Windows; `docs/manual.es.md` pasa a ser su traducción;
  - inicio rápido multiplataforma en el README y en las dos guías, probado de principio a fin en
    Linux (compilación, demo, clave, `survey`, `scan`, `reveal`, paquetes de reglas, `build.sh`);
  - SECURITY y las guías describen la lectura de WAR/EAR, keystores y certificados;
  - CONTRIBUTING refleja todos los paquetes y el flujo de generación de reglas;
  - el anexo para equipos bloqueados pasa a compilar desde fuentes;
  - la demo (`examples/demo-report`) se genera en inglés y `examples/repos.csv` apunta a ella;
  - NOTICE usaba aún el nombre antiguo del proyecto.
- **`keygen` crea la carpeta de la clave** si no existe (`--out ~/keys/team.key` fallaba la primera vez).

## 0.9.2 – 2026-11-05

Primera validación de la muestra contra el código real.

- **Campo `exclude` en las reglas**: descarta una coincidencia cuyo texto también encaja con otra
  expresión, al estilo de "pattern-not". Es dato de la regla, así que sirve para cualquier regla
  propia.
- **Credenciales en configuración y código más precisas.** La palabra clave (password, pwd,
  secret…) tiene que ser un segmento completo de la clave. Así `secretario` o `tokenizer` ya no
  cuentan. Tampoco cuentan los valores que son una URL o una ruta (`url.pwd.error=https://…`).

## 0.9.1 – 2026-11-04

Ajustes tras la primera ejecución de 0.9.0 en un entorno real.

- **Una autoridad de certificación nunca es persona física.** Algunas CA con correo o CIF en el
  sujeto se marcaban como personales y perdían el sujeto.
- **Hosts ocultos en todo el sujeto.** Un nombre de servidor dentro de una OU, una O o el alias de
  un certificado que no es CA se sustituye por `‹host›` y se cifra como el resto de hosts. Antes
  solo se cifraba si estaba en el CN.
- **Interfaz con varios proveedores.** Si varios sistemas implementan la interfaz que llama un
  consumidor (por ejemplo, un servicio de aviso que cada aplicación implementa), cada relación:
  - baja a confianza media;
  - indica "N sistemas implementan la interfaz";
  - añade la columna `candidateProviders` en `links.csv`.

  A cuál se llama lo decide la configuración en ejecución, no el código.

## 0.9.0 – 2026-11-04

Núcleo agnóstico: Traverse no lleva conocimiento de ningún país, sector u organización. Sirve igual
para cualquier proyecto; lo regional se añade, si se quiere, con paquetes de reglas.

- **Catálogo 2.0.0 por estándares**: las reglas `GOV-*` salen del catálogo embebido. En su lugar,
  firma electrónica por estándar: `SIGN-XMLDSIG`, `SIGN-CMS`, `SIGN-PDF`, `SIGN-ADES`
  (XAdES/CAdES/PAdES/JAdES, DSS), `SIGN-TIMESTAMP` (RFC 3161, antes `GOV-TIMESTAMP`), `SIGN-PKCS11`,
  y `AUTH-EIDAS` para identidad transfronteriza.
- **Paquetes de reglas opcionales** (`rulepacks/`): `es-public-sector.json` con @firma, Cl@ve, FIRe,
  Valida y FNMT (ids `ES-*`). Se cargan con `--rules` o con la nueva clave `"rules"` de
  `traverse.json`. Un test impide que un paquete sobrescriba una regla del núcleo.
- **El tipo de autenticación es dato de la regla** (`authKind`: user, service, signature,
  transport), no una tabla en el código: cualquier regla propia o de un paquete se coloca sola en
  la sección de seguridad.
- **Certificados de persona física por estándares**: atributos X.509 de nombre, apellidos y
  seudónimo, identificadores semánticos ETSI EN 319 412-1 (`PAS`, `IDC`, `PNO`, `TAX`, `TIN` + país),
  correo en un sujeto que no es un host o un CN con número de identificación. Se retiran los
  patrones propios de un país.
- **Vocabulario internacional**: más abreviaturas de entorno (prd, stg, uat, acc, hml, recette…) y
  nombres de carpeta contenedora en varios idiomas.
- Ejemplos, tests y documentación usan nombres neutros.

**Migración**: quien usaba las reglas `GOV-*` añade `--rules rulepacks/es-public-sector.json` (o
`"rules"` en `traverse.json`). Los ids cambian de `GOV-X` a `ES-X`; `GOV-TIMESTAMP` pasa a
`SIGN-TIMESTAMP` en el núcleo.

## 0.8.2 – 2026-11-04

- **"A quién llama" y "Quién le llama"**: nueva sección en cada informe y en el resumen ejecutivo. Incluye llamadas salientes agrupadas por aplicación, sistemas analizados que le llaman, datos compartidos e interfaces sin consumidor identificado (API, pantallas, otras).
- **Certificados públicos**:
  - lectura de ficheros PEM/DER (también sin extensión), entradas de confianza de truststores JKS y cadenas de las entradas con clave;
  - sujeto, emisor, caducidad, CA o identidad propia;
  - los certificados de persona física no guardan sujeto ni alias, y los CN con forma de host se cifran;
  - hallazgos `SEC-CERT-EXPIRED` y `SEC-CERT-EXPIRING`.
- **Enlace con context root versionado**: `/sign-gateway/signService` enlaza con el `/sign-gateway-2.3/signService` deducido del artefacto Maven (cliente de una pasarela de firma con mTLS).
- **Vistas por área**: `areas/<grupo>/` con portfolio, Markdown y diagramas cortados del portfolio global, incluyendo los sistemas relacionados de otras áreas. Enlaces desde el índice global.
- **`validation-sample.csv`**: 20 afirmaciones deterministas y estratificadas, con evidencia y columna de veredicto, para medir la precisión.
- Ajustes:
  - los entornos se deducen solo de rutas de configuración (un paquete `integracion` no es un entorno);
  - "login no visible" solo se indica en sistemas que exponen interfaces;
  - los despliegues GitOps no cuentan como llamadas;
  - los nombres de carpeta transliteran tildes.

## 0.8.1 – 2026-11-04

Primera ejecución completa en un entorno real (8 sistemas).

- **Terceros agrupados**: una fila por contraparte y endpoint, con los entornos que la declaran (int, pre, pro…) y el número de hosts, en lugar de una fila por host y entorno.
- **Contraparte por context root**: en las llamadas HTTP, el primer segmento de la ruta (`/sign-service/…`, `/docstore/…`) nombra la aplicación destino. Si coincide con un context root o módulo propio, se marca como "(propio sistema)" y sale del inventario de terceros.
- **Ámbito correcto**: las contrapartes nombradas por el código que no son sistemas del portfolio (esquemas ajenos, módulos, correo, servicios SOAP) aparecen como "externo".
- **URL que no son integraciones**: se ignoran los ficheros de mensajes (i18n), los de herramientas de build y CI (settings.xml de Maven, pipelines, deploy-request) y las URL de documentos (`.pdf`, `.xsd`…), repositorios (Artifactory, Nexus, SVN) e identificadores (eIDAS, OASIS…).
- **Tablas compartidas**: los nombres genéricos (ROL, USUARIO, AUDITORIA, CONFIGURACION…) sin esquema común ya no generan enlaces entre sistemas. FETCH, FIRST y similares ya no se toman como tablas.
- **SOAP**: el cliente generado y el servicio expuesto se enlazan aunque difieran en sufijos (`NotificationCallbackWSService` ≡ `notificationCallbackWS`). Un nombre de paquete que es variante del propio sistema ya no se toma como contraparte.
- **WAR**: el `pom.xml` embebido ya no duplica módulos. Nuevo diagnóstico `artifact-name-version` cuando el nombre del fichero y los metadatos internos no coinciden (p. ej. `-2.4-INT.war` construido como 2.3).
- **JAAS** usado en callbacks de un cliente SOAP con WS-Security se clasifica como autenticación entre servicios.
- **Estructura**: el proyecto raíz aparece como analizado y la lista omite las carpetas de imágenes y fuentes.

## 0.8.0 – 2026-11-04

Primeras ejecuciones en un entorno real y niveles de directorios de cinco repositorios
reales: el análisis tiene que acertar con dónde está el código a la primera, y todo lo que diga debe
salir del código, sin importar inventarios externos.

- **Reconocimiento de estructura** (`traverse survey` y paso previo de `scan`):
  - localiza el código, detecta copias del mismo proyecto y elige la línea principal (`trunk` → versión más alta → más reciente);
  - copias detectadas: mismas coordenadas de build o ≥ 80 % de rutas comunes; `branches/` y `tags/`, incluidos los anidados;
  - clasifica lo que no es código (documentación, modelos, binarios) y omite los runtimes incluidos en el repositorio (JDK, servidores);
  - `--layout all` (alias `--svn-all`) lo analiza todo;
  - nueva sección "Estructura del repositorio" en HTML, report.md y arc42, y línea en el resumen ejecutivo;
  - el sistema toma el nombre de la carpeta padre si la indicada es genérica (`…/orders/fuente` → `orders`).
- **Integraciones declaradas en configuración**:
  - URL en `.properties`, YAML, XML de Spring y descriptores de servidor (HTTP/SOAP, JDBC, LDAP, SMTP, FTP/SFTP, AMQP, outbound-socket-binding);
  - transporte y host cifrado; la clave sugiere la contraparte;
  - una URL base enlaza con el sistema del portfolio que expone rutas bajo ella.
- **Artefactos WAR/EAR de módulos sin código**: se leen por dentro (descriptores, configuración, librerías embebidas para SBOM y vulnerabilidades) y se avisa si la versión no coincide con la del descriptor (`artifact-version-mismatch`).
- **Autenticación por tipo** (`Finding.authKind`: user / service / signature / transport):
  - nuevas reglas: login propio (`AuthenticationProvider`…), SSO preautenticado, autenticación de contenedor, Keycloak (configuración y `auth-method`), mTLS y features de seguridad de Liberty;
  - "Login de usuarios: no visible en el código" cuando no hay evidencia.
- **API frente a pantallas** en las interfaces expuestas (`Contract.role`), en tablas y resumen.
- **Contrapartes desde el código** (`Contract.counterpart`): paquete del cliente, servicio WSDL, clave de configuración, JNDI, esquemas de BD de otros sistemas y módulos de plataforma propios.
- **Menos ruido**:
  - los alias y rutas JPQL (`join u.roles`) ya no se toman como `esquema.tabla`, y se leen las listas `FROM a, b`;
  - keystores con clave privada en severidad alta y truststores con solo certificados públicos como info (se inspeccionan las entradas sin descifrar); se reconocen las copias `.jksOLD`/`.p12.bak`;
  - la evidencia mostrada prioriza el código sobre la configuración de logging.
- Esquema de hechos 1.5.0 (aditivo); reglas 1.3.0 (160 reglas).

## 0.7.0 – 2026-11-04

Renombrado de OAT a **Traverse**: nombre de proyecto más genérico e internacional (sigue siendo lo que era, un analizador de arquitectura estático y offline). Sin cambios de comportamiento.

- Module path Go: `github.com/oat-toolkit/oat` → `github.com/oskillarrr/traverse`.
- Binario y comando CLI: `oat`/`oat.exe` → `traverse`/`traverse.exe` (`cmd/oat` → `cmd/traverse`).
- Variables de entorno: `OAT_LANG`, `OAT_KEY_FILE`, `OAT_PASSPHRASE`, `OAT_VULNDB` → `TRAVERSE_LANG`, `TRAVERSE_KEY_FILE`, `TRAVERSE_PASSPHRASE`, `TRAVERSE_VULNDB`.
- Fichero de configuración opcional: `oat.json` → `traverse.json`.
- `ToolInfo.Name`, cabeceras SARIF/CycloneDX, pies de página HTML, comentarios de ficheros generados (Mermaid, Structurizr DSL, PlantUML) y la clave `partialFingerprints` de SARIF (`oatFindingId/v1` → `traverseFindingId/v1`) actualizados.
- Documentación (README, manual, CONTRIBUTING, SECURITY), pipelines de CI (`.gitlab-ci.yml`, `.github/workflows/ci.yml`) y scripts de build/validación renombrados.
- `schemaVersion` de hechos sin cambios (1.4.0): es un rename puro, no una versión de datos nueva.

## 0.6.0 – 2026-11-04

Autenticación, mTLS e inventario de terceros, a partir de la revisión de un arquitecto del equipo; ver `openspec/changes/add-auth-and-integration-inventory`.

- **25 reglas nuevas de autenticación y transporte** (152 en total): Spring Security por método, X.509/certificado de cliente, JWT, OAuth2/OIDC, SAML, CAS, Keycloak, Kerberos/SPNEGO, LDAP/AD, WS-Security, API key, Basic/Bearer en clientes, keystore de cliente (mTLS), TLS propio, verificación TLS deshabilitada, protocolos TLS obsoletos, TLS y mTLS de Spring Boot; y del ecosistema público español: @firma/AutoFirma, Cl@ve/eIDAS, a central signature service, Valida/DSS, FNMT/CERES y sellado de tiempo.
- **Descriptores**: `CLIENT-CERT` y `transport-guarantee` en web.xml; conectores TLS/clientAuth de Tomcat (`server.xml`); `https-listener verify-client` y `security-realm` de WildFly; protocolos TLS débiles en conectores.
- **Transporte por integración**: cada contrato HTTP/SOAP lleva `transport` (TLS / sin cifrar) según el esquema de la URL.
- **Informe**: sección "Autenticación y seguridad de transporte" (mecanismos, mTLS, puntos débiles: HTTP sin cifrar, SOAP sin WS-Security, TLS deshabilitado u obsoleto), **inventario de sistemas de terceros**, columnas de transporte y auth en integraciones, y líneas de seguridad y terceros en el resumen ejecutivo. En HTML y report.md.
- Esquema de hechos 1.4.0 (aditivo).

## 0.5.0 – 2026-11-04

Mejoras tras verificar Traverse con un proyecto legacy real (Java/JBoss, iBATIS + Struts 2 + Spring); ver `openspec/changes/add-ibatis-struts-and-markdown`.

- **iBATIS/MyBatis**: se extraen las tablas del SQL de los mappers XML (reconocidos por elemento raíz o DOCTYPE, con cualquier nombre), incluido el SQL dentro de etiquetas dinámicas. El namespace del mapper queda como componente. Antes el modelo de datos de estas aplicaciones era casi invisible.
- **Struts 2 y 1 por contenido**: las acciones definidas en ficheros incluidos con nombre arbitrario se reconocen por el elemento raíz `<struts>` / `<struts-config>` y producen endpoints HTTP entrantes.
- **Múltiples raíces web**: un módulo con más de un `WEB-INF/web.xml` emite el diagnóstico `multiple-web-roots`.
- **Resumen ejecutivo** en el informe HTML (por sistema y portfolio), en `report.md` y en `portfolio.md`.
- **Exportación a Markdown**: nuevo `report.md` por sistema, informe analítico poblado equivalente a `report.html`.
- Refactor interno: `internal/sqlparse` comparte la extracción de tablas entre el escáner de código y el de descriptores.

## 0.4.1 – 2026-11-03

- Fix (found in the first Windows run with a real OSV dump): an ecosystem was treated as loaded when it only appeared in advisories of another ecosystem's dump (e.g. 67 npm entries inside `Maven/all.zip`), so its libraries were checked against an incomplete list. Now an ecosystem counts as loaded only when its own dump is present; the others are reported as `ecosystem-not-loaded`. `traverse vulndb info` marks each ecosystem as `complete` or `partial`.
- Demo vulnerability database split into one zip per ecosystem, like the real dumps.
- First run on Windows (amd64) confirmed: `version`, `vulndb info` with the real Maven dump.

## 0.4.0 – 2026-11-03

Decision D4 (option B) and confirmation of D6; see `openspec/changes/add-offline-vulnerability-matching`.

- **Offline vulnerability matching** with OSV dumps: `--vulndb` / `TRAVERSE_VULNDB`, `traverse vulndb info`; Maven, npm, NuGet, PyPI, Packagist and Go; ecosystem-specific version ordering (Maven ComparableVersion, SemVer 2.0, NuGet, PEP 440) and OSV range semantics (`introduced`, `fixed`, `last_affected`, explicit versions); CVSS v3.x base score computed with the FIRST formula, source severity otherwise; malicious packages (`MAL-`) as critical; aliases unified; withdrawn advisories ignored; stale-database warning after 30 days.
- **Lockfiles** pin exact versions of direct dependencies and add transitive packages: package-lock.json / npm-shrinkwrap.json (v1–v3), yarn.lock (v1 and Berry), pnpm-lock.yaml (v5–v9), bun.lock, composer.lock, packages.lock.json, poetry.lock, uv.lock, Pipfile.lock, gradle.lockfile; .NET central package management (`Directory.Packages.props`, with `$(...)` properties).
- **Committed JARs** identified by their embedded `META-INF/maven/**/pom.properties`.
- Versions that cannot be evaluated are listed with a reason (`external-bom`, `range`, `unresolved`, `missing`, `ecosystem-not-loaded`), never treated as safe.
- Outputs: `vulnerabilities` in CycloneDX 1.6 SBOMs (validated with the official schema), `VULN/<id>` rules in SARIF, HTML sections, arc42 section 11.1, portfolio document section 8, `vulnerabilities.csv`, `vulnerabilities-unevaluated.csv`; SBOMs include lockfile transitives.
- Composer platform requirements (`ext-*`, `lib-*`) are no longer reported as library dependencies.
- `scripts/fetch-osv.sh` and `scripts/fetch-osv.ps1` to download the dumps on an authorised machine; synthetic demo database in `examples/demo-vulndb`.
- Facts schema 1.3.0 (additive).

## 0.3.0 – 2026-11-03

Decisions D1–D3, D5, D6 of the verification (see `openspec/changes/add-deployment-arc42-hosts`).

- **arc42 documentation in Markdown** per system (12 sections, pre-filled, ✍️ markers) and **portfolio document** with integration matrix and catalog; integration catalog with direction, counterpart, mode (sync/async/shared data/build/deployment), component, evidence and confidence; statically derived runtime scenarios; Mermaid diagrams validated with Mermaid CLI.
- **Deployment view**: Dockerfile, Docker Compose, Kubernetes, OpenShift and Helm with a dependency-free YAML reader; links Ingress→Service→workload→ConfigMap/Secret, Compose depends_on and env references; module linking via build context or image name; cross-repository `deploys` relationships (GitOps repositories); deployment risks; C4 deployment environments (validated with Structurizr CLI); ArchiMate Node/Artifact/Technology Interface.
- **Encrypted host names**: `traverse keygen`, `--key-file`/`TRAVERSE_KEY_FILE`/`TRAVERSE_PASSPHRASE`, HMAC pseudonyms in every output, AES-256-GCM vault, `traverse reveal`.
- **Low-confidence relationships**: attenuated in SVG, Mermaid, Structurizr and PlantUML; `--min-confidence` filter for diagrams.
- Robustness: per-file recovery turns extractor errors into diagnostics; nil-safe YAML access.
- Determinism: map-order fixes in DSL, ArchiMate and CSV; test asserting byte-identical outputs across runs.
- Fixed missing translation keys (now covered by a test).
- Requires Go 1.24 to build (standard-library HKDF/PBKDF2).

## 0.2.0 – 2026-11-03

Verification against 13 real open-source projects and official standards.

- **New extractors**: Kotlin and C# structural parsing (shared with Java); ASP.NET attribute/conventional routing, minimal APIs and Razor Pages; JS/TS, Python, PHP, Go and Ruby modules, imports, HTTP routes and ORM tables; mainframe COBOL/CICS/JCL/BMS/CSD/HLASM (programs, copybooks, transfers, screens, files, datasets, transactions).
- **New standard exports**: ArchiMate 3.1 Open Exchange Format, SARIF 2.1.0, CycloneDX 1.6 SBOM — all validated against official schemas.
- **Fixes found on real code**: artifact-repository rule no longer fires on license URLs; test sources no longer produce contracts; URL literals only count as calls when the file has an HTTP client; shared tables require same schema or datasource for medium confidence; calls matching an endpoint of the same system are internal; WAR context roots inferred from finalName/artifact; servlet mappings prefixed with context root; `.ctl`/`.prc` disambiguated by content; Structurizr DSL valid for repositories without build files; Laravel nested resources.
- Layer vocabulary extended for .NET, web frameworks and clean-architecture names.


## 0.1.0 – 2026-11-03

First public version, replacing the PowerShell proof of concept.

- Single self-contained binary for Windows, Linux and macOS (amd64/arm64); Go standard library only.
- Multi-repository portfolio analysis with manifest (`repos.csv`).
- Cross-repository links: libraries, HTTP, SOAP, JMS/Kafka, EJB, shared tables and datasources; version divergence; external dependencies.
- Java structural parser with comment/string-aware lexer: nested types, annotation-to-declaration binding, multi-line annotations, wildcard imports, same-package references, injection.
- Layer classification by stereotype, inheritance, package token (English/Spanish) and naming, with explanation and configurable overrides; layer violations; package metrics; cycles (Tarjan).
- Build manifests: Maven (parent inheritance, properties, dependencyManagement), Ant, Ivy, Gradle, npm, Composer, MSBuild/NuGet, Python, Go; vendored JARs.
- JEE/JBoss/Spring/Struts/WSDL/server descriptors parsed as XML (no DTD resolution).
- 127 data-driven rules with self-tested examples; custom rule files.
- Subversion standard layout awareness; legacy encodings.
- Reports: HTML (light/dark, offline, filterable), CSV, JSON, SVG, Mermaid, Structurizr DSL, C4-PlantUML; English and Spanish.
- Fixes all 15 defects found in the PoC review (see `openspec/changes/archive/`).
