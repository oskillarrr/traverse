# Verificación de cobertura y formatos — 2026-11-03

## 1. Método

1. **13 proyectos open source reales** clonados de GitHub (`--depth 1`), elegidos para cubrir tecnologías y épocas distintas, y analizados como un único portfolio.
2. **Revisión manual de muestras**: endpoints, capas, dependencias, secretos, contratos y enlaces entre sistemas, contrastados con el código fuente.
3. **Validación de cada formato de salida** con la herramienta o el esquema oficial (`scripts/validate-outputs.sh`).
4. **Contraste** de lo extraído con lo que pide la documentación de arquitectura de referencia (arc42, ISO/IEC/IEEE 42010, modelo C4, catálogos TOGAF).

## 2. Proyectos y resultados (antes → después de esta verificación)

| Proyecto | Tecnología | Componentes | Dependencias | Interfaces expuestas | Tablas/datasets | Riesgos A+M | % sin capa |
|---|---|---|---|---|---|---|---|
| angular-realworld-example-app | Angular/TS | 0 → **44** | 0 → **102** | 0 | 0 | 0 | 2 % |
| aws-mainframe-modernization-carddemo | COBOL/CICS/JCL/BMS/DB2 | 0 → **190** | 0 → **319** | 0 → **28** | 3 → **101** | 0 | 0 % |
| bakerydemo | Python/Django | 0 → **113** | 0 → **31** | 0 → **13** | 1 | 3 | 8 % |
| cargotracker | Jakarta EE | 106 → **113** | 323 | 4 | 7 | 3 | 0 % |
| eShopOnWeb | .NET 8 (MVC, Razor, Blazor, minimal API) | 0 → **210** | 0 → **368** | 0 → **45** | 0 | 4 | 13 % |
| golang-gin-realworld-example-app | Go/gin | 0 → **4** | 0 → **15** | 0 → **21** | 4 → **2** | 0 | 75 % |
| jboss-eap-quickstarts | Java EE/JBoss EAP (74 módulos) | 213 → **221** | 209 | 68 | 21 → **20** | 310 → **243** | 15 % |
| jpetstore-6 | Java/Stripes/MyBatis | 26 | 48 | 23 | 18 | 5 | 0 % |
| koel | PHP/Laravel + Vue/TS | 0 → **1880** | 0 → **3265** | 0 → **150** | 11 → **42** | 3 | 11 % |
| node-express-realworld-example-app | Node/Express/TS | 0 → **32** | 0 → **40** | 0 → **20** | 8 | 0 | 9 % |
| spring-petclinic | Spring Boot | 25 | 40 | 17 | 7 | 3 | 12 % |
| spring-petclinic-kotlin | Kotlin/Spring Boot | 0 → **24** | 0 → **37** | 0 → **18** | 10 | 2 | 8 % |
| spring-petclinic-microservices | Spring Cloud (9 servicios) | 59 → **79** | 59 → **98** | 15 | 7 | 1 | 5 % |

Tiempo total del portfolio (≈ 6.000 ficheros): unos 5 s.

## 3. Defectos encontrados y corregidos

| # | Defecto (evidencia) | Corrección |
|---|---|---|
| 1 | `OPS-INSECURE-REPO` saltaba con las URL de licencia del pom (83 falsos positivos en jboss-eap-quickstarts) | Solo dentro de `<repository>`, `<pluginRepository>`, `<snapshotRepository>`, `<mirror>` y `maven { url }` de Gradle |
| 2 | Llamadas HTTP extraídas de ficheros de test (404 en koel) | Los contratos de tests se excluyen salvo con `--include-tests` |
| 3 | URL de contenido (YouTube en migraciones de Django) contadas como integraciones | Una URL solo cuenta como llamada si el fichero usa un cliente HTTP |
| 4 | Tablas con el mismo nombre en sistemas independientes (USERS) enlazadas con confianza media | Confianza baja si solo coincide el nombre; media si coincide el esquema o el datasource (`basis`) |
| 5 | Llamadas entre microservicios del mismo repositorio enlazadas con otros sistemas | Si la llamada coincide con un endpoint propio, es interna |
| 6 | Coincidencias HTTP que solo casan mediante variables de ruta con confianza media | Confianza baja |
| 7 | `@WebServlet` y `servlet-mapping` sin context root; WAR sin `jboss-web.xml` con "context root unknown" | Prefijo de context root; inferencia desde `finalName` o `artifactId-version` (marcada como inferida) |
| 8 | `.ctl` de DB2 detectados como VB6; `.bms` y `.prc` de JCL sin reconocer | Desambiguación por contenido; BMS, HLASM, REXX, CSD y Prisma registrados |
| 9 | Clases `@Remote` sin capa; paquetes con `_` no tokenizados | Estereotipos `Remote`/`Local`; tokenización por `.` y `_` |
| 10 | Structurizr DSL **inválido** en repositorios sin fichero de build (detectado por Structurizr CLI) | Contenedor sintético para fuentes sin descriptor |
| 11 | Números de secuencia COBOL (columnas 73-80) tomados como nombres de programa | Áreas de secuencia e identificación ignoradas (formato fijo) |
| 12 | `EXEC CICS XCTL PROGRAM(variable)` sin resolver (3 de 28 transferencias) | Resolución por `VALUE` y por `MOVE 'X' TO variable`; las no resolubles van a `diagnostics.csv` |
| 13 | Servicios del sistema (CEE*, MQ*, DSN*) como "programas externos" | Filtrados como utilidades del runtime |
| 14 | `RETURN TRANSID` del propio sistema mostrado como dependencia de salida | Las llamadas a recursos propios no se dibujan en el contexto |
| 15 | `Route::apiResource('albums.songs')` como `/albums.songs` | `/albums/{album}/songs` |

## 4. Huecos cubiertos en esta verificación

| Tecnología | Antes | Después |
|---|---|---|
| C# / .NET | Solo inventario y reglas | Clases, capas, dependencias (`using`), rutas por atributo (`[controller]`, `~/`), rutas convencionales MVC, minimal APIs, Razor Pages, Web Forms/ASMX/WCF |
| Kotlin | Solo inventario | El mismo análisis estructural que Java (Spring, JPA, endpoints) |
| JS/TS, Python, PHP, Go, Ruby | Solo inventario | Módulos, imports resueltos, rutas (Express, NestJS, Flask, FastAPI, Django, Laravel, Symfony, gin/echo/chi, Rails) y tablas ORM |
| Mainframe | Solo inventario y reglas | Programas, copybooks, jobs, procs, mapsets; CALL/LINK/XCTL; mapas BMS; ficheros CICS → datasets (vía CSD); datasets JCL con DISP; transacciones CSD; colas TS/TD; EXEC SQL |
| Formatos estándar | C4 (Structurizr, PlantUML), Mermaid | + ArchiMate 3.1 Exchange, SARIF 2.1.0, CycloneDX 1.6 |

## 5. Validación de formatos con herramientas y esquemas oficiales

| Salida | Estándar | Validador | Resultado (13 sistemas + demo) |
|---|---|---|---|
| `workspace.dsl` (×14) | Modelo C4 / Structurizr DSL | Structurizr CLI 2024.12.07 `validate` y `export -format plantuml/c4plantuml` | ✅ tras corregir el defecto 10 |
| `landscape.puml` | C4-PlantUML | PlantUML 1.2024.8 (librería C4 incluida, sin red) | ✅ |
| `architecture.archimate.xml` | ArchiMate 3.1 Model Exchange File Format (The Open Group) | `xmllint` con `archimate3_Diagram.xsd` | ✅ (311 elementos, 341 relaciones) |
| `findings.sarif` (×14) | SARIF 2.1.0 (OASIS Standard) | JSON Schema oficial `sarif-schema-2.1.0.json` | ✅ |
| `sbom.cdx.json` (×13) | CycloneDX 1.6 (ECMA-424) + Package URL | JSON Schema oficial `bom-1.6.schema.json` | ✅ |
| `*.mmd` (×14) | Mermaid | Mermaid CLI | ✅ |

La correspondencia con ArchiMate respeta las relaciones permitidas por el metamodelo 3.1:

- *Serving*: System Software → Application Component, y Component → Component.
- *Realization*: Artifact → Component.
- *Composition*: Component → Interface.
- *Access*: Component → Data Object, con `accessType`.
- *Flow*: Component → Component.
- *Aggregation*: Grouping → Component.
- *Association*.

## 6. Cobertura frente a lo que necesita un arquitecto

### arc42

| Sección | Cobertura | Fuente en Traverse |
|---|---|---|
| 1 Introducción y objetivos | ❌ Requiere personas | — |
| 2 Restricciones | 🟡 Técnicas | Runtimes, plataformas, frameworks, versiones de Java/.NET |
| 3 Contexto y alcance | ✅ | Diagrama de contexto, interfaces expuestas y consumidas, dependencias externas, C4 contexto |
| 4 Estrategia de solución | 🟡 | Tecnologías y frameworks dominantes (matriz) |
| 5 Vista de bloques | ✅ | Contenedores (módulos/desplegables), componentes por capa, dependencias, métricas, ciclos |
| 6 Vista de ejecución | 🟡 Parcial | Cadenas estáticas: JCL → programa → programa, transacción CICS → programa, MDB ← cola |
| 7 Vista de despliegue | 🟡 Parcial | WAR/EAR, context roots, módulos JBoss, datasources. **No** se modelan Docker, Compose ni Kubernetes (solo se detectan) |
| 8 Conceptos transversales | 🟡 | Seguridad, persistencia, transacciones, logging, caché y planificación (reglas) |
| 9 Decisiones | ❌ Requiere personas | — |
| 10 Calidad | ❌ / 🟡 | Indicadores indirectos: acoplamiento, ciclos, violaciones de capas |
| 11 Riesgos y deuda técnica | ✅ | Riesgos priorizados, EOL, migración javax→jakarta, .NET Framework, secretos (ubicación) |
| 12 Glosario | ❌ | — |

### Catálogos TOGAF / arquitectura empresarial

| Artefacto | Cobertura |
|---|---|
| Catálogo de aplicaciones | ✅ `systems.csv`, ArchiMate |
| Catálogo de interfaces | ✅ `contracts.csv`, `links.csv` |
| Matriz aplicación × datos | ✅ Acceso a datos, tablas compartidas, ArchiMate *Access* |
| Catálogo tecnológico / aplicación × tecnología | ✅ Matriz de tecnologías, SBOM |
| Matriz aplicación × organización / proceso | ❌ No está en el código: se puede añadir con `group` en el manifiesto |

## 7. Lo que el análisis estático no puede dar

- Comportamiento en ejecución (flujos reales, volúmenes, latencias).
- Configuración que no está en el repositorio: URL por entorno, balanceadores, ESB/DataPower, colas definidas solo en el servidor.
- Reflexión, inyección por nombre en ejecución, rutas o SQL construidos dinámicamente.
- Decisiones, objetivos y requisitos de calidad: arc42 §1, §9, §10 y §12.

## 8. Puntos abiertos para decidir (ver proposal.md)

D1. Qué se entiende por "convenciones oficiales": estándares internacionales (implementados) o una plantilla o metodología de la organización (Métrica v3, plantilla propia, ENS).
D2. Documento de arquitectura generado: arc42 prerrellenado (Markdown/AsciiDoc), plantilla corporativa o ninguno.
D3. Vista de despliegue: modelar Dockerfile, Compose y Kubernetes. Requiere un lector YAML propio para mantener cero dependencias.
D4. Vulnerabilidades: cruzar el SBOM con una copia offline de OSV/NVD o delegarlo en Dependency-Track.
D5. Nombres de host por defecto (`--include-hosts`): mejoran el catálogo de dependencias externas, pero exponen infraestructura.
D6. Política de enlaces de baja confianza en los diagramas: mostrarlos atenuados, ocultarlos o hacerlo configurable.
