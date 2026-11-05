# Design — Traverse v2

> **Archivado 2026-11-03.** La decisión D1 (extractor en PowerShell) quedó **sustituida**: se requirió una herramienta portable, local y agnóstica de tecnología y sistema operativo, por lo que se implementó en Go (binario único, solo librería estándar). Ver `openspec/project.md` → ADR-001. El resto del diseño (pipeline, esquema de hechos, contratos, matching) se implementó tal cual.

## Context
- Portfolio de N aplicaciones Java EE/JBoss en repos independientes, con librerías comunes
  compartidas por Maven (p. ej. `comun-dto`), integración por REST/SOAP/JMS/JNDI y,
  previsiblemente, BBDD compartidas.
- Entorno corporativo bloqueado: PS 5.1 garantizado; JDK probablemente presente (equipos Java) pero su
  uso para herramientas internas debe confirmarse con Sistemas.
- Sin IA: toda inferencia debe ser **determinista, explicable y reproducible**.

## Goals / Non-Goals
**Goals**
- Vista por aplicación (C4 nivel 2–3) y vista de portfolio (C4 nivel 1 / landscape).
- Cada afirmación del informe enlaza a evidencia `repo:file:line` y a una regla versionada.
- Re-ejecutable: mismo input + misma versión de reglas ⇒ mismo output (orden estable).
- Validación humana acumulativa.

**Non-Goals**
- Reconstruir la topología de producción (balanceadores, DataPower, certificados).
- Análisis semántico Java completo (resolución de tipos con classpath, reflexión, AOP).
- Estimación automática de esfuerzo de migración.

## Decisions

### D1 — Lenguaje del extractor: contrato primero, implementación sustituible
| Opción | Fidelidad | Dependencias | Aprobación | Comentario |
|---|---|---|---|---|
| A. PowerShell 5.1 mejorado | Media | Ninguna | Ya resuelta | Techo claro: sin AST real |
| B. CLI Java propia + JavaParser (+ symbol solver) | Alta | 1–3 jars vendorizados con checksum | Requiere JDK y alta de librería | Resuelve tipos sin import, anidadas, genéricos, anotaciones multilínea |
| C. Herramienta existente (jQAssistant, MTA/Windup) + agregador propio | Alta (reglas maduras) | Distribución completa | Mayor esfuerzo de homologación | MTA ya trae reglas de migración JBoss/EAP y javax→jakarta |
| D. `jdeps` (incluido en el JDK) sobre WAR/EAR ya construidos | Alta en dependencias de clase | Ninguna extra si hay JDK | Mínima | Necesita binarios del repositorio de artefactos; no da endpoints |

**Decisión**: implementar **A** ya, con el esquema `facts.json` como contrato. Abrir en paralelo la
consulta a Sistemas por **D** (coste ~0, gran ganancia en el grafo de dependencias) y por **B**.
Agregación e informes no dependen del extractor elegido. Se descarta reescribir todo en Java
de inicio: bloquearía el proyecto en la homologación.

### D2 — Pipeline de tres etapas con artefactos intermedios
```
repos.csv ──► extract (por repo, idempotente, cacheable por hash)
                 └─► out/<repo>/facts.json  (+ CSV derivados)
           ──► aggregate (portfolio)
                 └─► out/_portfolio/portfolio.json
           ──► render (por repo + portfolio)
                 └─► HTML, SVG, .mmd, .dsl (Structurizr), .puml (C4-PlantUML), CSV
```
Cada etapa se puede re-ejecutar aislada. `ejecutar.ps1` conserva la interfaz de la PoC
(extract+render de un repo).

### D3 — Esquema de hechos (`facts.json`, `schemaVersion` semver)
```jsonc
{
  "schemaVersion": "2.0.0", "tool": {"version": "2.0.0", "rulesVersion": "1.0.0"},
  "repo": {"name": "", "rootHash": "", "scannedAt": ""},
  "modules":   [{"id": "g:a", "groupId": "", "artifactId": "", "version": "", "packaging": "war",
                 "parent": "g:a", "path": "modulo-a", "dependsOn": [{"gav": "g:a:v?", "scope": "", "managed": false}]}],
  "deployables": [{"module": "g:a", "kind": "WAR|EAR|EJB-JAR", "contextRoot": "/x",
                   "applicationPath": "/api", "earModules": []}],
  "components": [{"fqn": "p.Outer.Inner", "kind": "class", "module": "g:a", "sourceSet": "main|test",
                  "stereotypes": ["Stateless","Path"], "layer": "Negocio", "layerReason": "token:servicio",
                  "file": "", "line": 0}],
  "dependencies": [{"from": "fqn", "to": "fqn", "kind": "import|field|extends|implements|inject|new|samePackageRef",
                    "confidence": "alta|media|baja", "file": "", "line": 0}],
  "contracts": [{"direction": "provided|consumed", "kind": "http|soap|jms|jndi|db-table|ejb-remote",
                 "key": "GET /x/api/expedientes/{id}", "normalizedKey": "GET /x/api/expedientes/*",
                 "component": "fqn", "file": "", "line": 0, "confidence": ""}],
  "findings": [{"id": "sha1(ruleId|repo|file|anchor)", "ruleId": "JBOSS-001", "category": "", "severity": "",
                "confidence": "", "file": "", "lines": [12, 40], "count": 2, "sourceSet": "main"}],
  "diagnostics": [{"file": "", "kind": "read-error|too-big|encoding-fallback|xml-invalid"}]
}
```
`anchor` del id de hallazgo = FQN del componente o ruta XPath, **no el número de línea**, para
que la huella sobreviva a ediciones del fichero.

### D4 — Pre-pase léxico Java (sin AST)
Máquina de estados de un único recorrido que reemplaza comentarios `//`, `/* */` y literales
`"..."`, `'.'`, text blocks `"""` por espacios **preservando saltos de línea**. Las reglas de
tecnología se aplican sobre el texto limpio; los literales se conservan aparte solo para
extraer rutas (`@Path("...")`) y nombres de destinos JMS/JNDI.
Sobre el texto limpio:
- Pila de llaves para determinar anidamiento → FQN correcto de clases internas.
- Bloque de anotaciones = secuencia contigua de `@Ident(...)` (paréntesis balanceados,
  multilínea) inmediatamente anterior a una declaración. Se asocia a **esa** declaración y a
  ninguna otra (elimina el bug de la ventana de 10 líneas).
- Índice de offsets de línea + búsqueda binaria (sustituye `Count-Line`, O(n) por match).

### D5 — Parseo XML con `System.Xml`
`XmlReaderSettings { DtdProcessing = Ignore; XmlResolver = $null; MaxCharactersFromEntities = 0 }`.
Se namespace-agnostiza con `local-name()`. Cubre `pom.xml`, `web.xml`, `persistence.xml`,
`ejb-jar.xml`, `application.xml`, `jboss-web.xml`, `jboss-deployment-structure.xml`,
`jboss-ejb3.xml`, `faces-config.xml`, `struts-config.xml`, `*-ds.xml`, WSDL, Spring XML.
Si el XML es inválido: fallback a regex con `confidence=baja` y diagnóstico.

### D6 — Resolución Maven ligera (sin Maven)
- GAV propio con herencia de `groupId`/`version` del `<parent>`.
- Propiedades `${...}` resueltas dentro del reactor (pom actual → padres presentes en el repo);
  si no se resuelve, se conserva el literal y `resolved=false`.
- `<dependencyManagement>` y dependencias de `<plugin>` se registran aparte (`managed=true`) y
  **no** generan aristas.
- `<modules>` → árbol de módulos; `packaging` → tipo de desplegable.

### D7 — Contratos y matching entre repos
- **HTTP provisto**: `contextRoot` + `@ApplicationPath`/`servlet-mapping` + `@Path` clase +
  `@Path` método. Normalización: variables `{x}` → `*`, sin barra final, minúsculas en verbo.
- **HTTP consumido**: literales de ruta (no hosts) en clientes (`WebTarget.path`, `RestTemplate`,
  `@RegisterRestClient`, `@FeignClient`) y **claves** de propiedades `*.url|*.endpoint|*.uri`
  (solo la clave, nunca el valor). Matching por sufijo de ruta normalizada → `confidence=media`.
- **JMS / JNDI / EJB remoto**: nombres de destino y de binding como clave exacta.
- **BD**: `@Table`/`@Entity`, `@NamedQuery`/JPQL `FROM X`, SQL `FROM|JOIN|INTO|UPDATE <tabla>` en
  `.sql`/literales → matriz aplicación×tabla (lectura/escritura cuando sea inferible).
- **Maven**: `consumed` = dependencia; `provided` = GAV del módulo. Una librería compartida es un
  nodo propio en el landscape.

### D8 — Capas por tokens configurables
Se divide el paquete en segmentos (`split('.')`) y se compara **por segmento completo** contra
un diccionario ES/EN (`rest, web, controller, controlador, recurso, servlet, action(s)` …
`service, servicio, negocio, business, facade, fachada, ejb` … `dao, repository, repositorio,
persistencia` …). Prioridad: override de config > estereotipo por anotación del propio tipo >
token de paquete más profundo > `Otros`. Se registra `layerReason`.
La capa se calcula **por tipo**, no por fichero (un `@Entity` en `modelo` es Modelo, no
Persistencia: la anotación indica estereotipo, no capa).

### D9 — Reglas como datos
`rules/tecnologias.psd1` (cargado con `Import-PowerShellDataFile`, disponible en 5.1):
```powershell
@{ Id='JMS-001'; Version='1.0.0'; Category='JMS'; Applies=@('.java');
   Pattern='\bjavax\.jms\.|\bjakarta\.jms\.|@JmsListener\b'; Confidence='media';
   Risk='JMS: revisar broker, colas y conexión'; Severity='media';
   Positive=@('import javax.jms.Queue;'); Negative=@('// import javax.jms.Queue;') }
```
Los ejemplos `Positive/Negative` se ejecutan como tests automáticamente.

### D10 — Métricas de arquitectura
Por paquete y por módulo: Ca, Ce, inestabilidad I = Ce/(Ca+Ce), nº tipos, LOC.
Ciclos con Tarjan (SCC) a nivel paquete, módulo y repo. Violaciones de capa según matriz
permitida configurable (por defecto: Web→Negocio→Persistencia, Modelo accesible desde todas).

### D11 — Rendimiento
- Poda de directorios excluidos **durante** el recorrido (`[IO.Directory]::EnumerateDirectories`
  recursivo propio), no filtrado posterior de `Get-ChildItem -Recurse`.
- Una regex alternada compilada por extensión (`RegexOptions.Compiled`) en vez de 20 `-match`
  por línea; se evalúa sobre el texto completo y se mapea offset→línea.
- `List[T]` y `StringBuilder` en todas las salidas; nada de `+=` sobre strings en bucles.
- Caché por hash SHA-256 de fichero (opcional) para ejecuciones incrementales.
- Referencia medida en PoC 1.0 (PS 7, Linux): 4,8 MB / 300 ficheros ≈ 10 s. Objetivo v2 en PS 5.1:
  ≤ 2 min para 100 MB de fuente.

## Risks / Trade-offs
- **Regex + léxico sigue sin resolver tipos** → aristas `samePackageRef` y `field` con
  confianza media; mitigación D1-D/B.
- **Matching de contratos por cadena** genera falsos enlaces con rutas genéricas (`/api/*`) →
  umbral de especificidad mínima (≥ 2 segmentos literales) y marca `confidence`.
- **Más artefactos de salida = más superficie de información sensible** → marca de
  clasificación, `-IncluirHosts` sigue opt-in, sin valores de propiedades.
- **Ruptura de formato** → `-CompatV1` una versión.

## Migration Plan
1. v1.1 (rápido, sin cambio de formato): corregir bugs de `evidence/REVISION-PoC-v1.md`.
2. v2.0-alpha: módulo + facts.json + extractor léxico/XML + Pester.
3. v2.0-beta: agregación de portfolio + C4 + validaciones persistentes.
4. v2.0: piloto con 3 repos reales relacionados (una librería común + dos consumidoras),
   medición de precisión/recall sobre muestra validada.

## Open Questions
- ¿Está permitido ejecutar `jdeps`/`java -jar` con el JDK ya instalado en el entorno corporativo?
- ¿Hay acceso de lectura a los WAR/EAR del repositorio de artefactos (Nexus/Artifactory)?
- ¿Existen exportaciones de `standalone.xml`/`domain.xml` de los servidores para cerrar los
  JNDI/datasources declarados?
- ¿Qué herramienta de diagramas está homologada (Structurizr Lite on-prem, PlantUML jar, draw.io)?
