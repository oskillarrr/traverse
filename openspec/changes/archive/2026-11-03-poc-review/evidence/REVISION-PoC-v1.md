# Revisión técnica — Traverse PoC 1.0

Ejecución real de `ejecutar.ps1` con PowerShell 7.4.6 (Linux) sobre:
1. `ejemplo-publico` incluido → **todos los criterios de VALIDACION.md se cumplen**.
2. `fixtures/legacy-es` (este change) → casos realistas de código legacy en español.
3. Repo sintético de 300 ficheros / 4,8 MB → **10 s** (PS 5.1 será previsiblemente más lento).

Pendiente: repetir en Windows PowerShell 5.1 dentro del entorno corporativo.

## Lo que está bien (mantener)
- Garantías de seguridad claras: solo lectura, salida fuera del repo, sin red, sin valores de
  propiedades, hosts opt-in, CSP en el HTML, avisos de alcance honestos.
- Trazabilidad fichero:línea en todos los CSV.
- Entregables útiles para Excel (CSV) + visual (SVG autocontenido) + editable (Mermaid).
- Vocabulario de validación (confirmado / probable / pendiente).

## Defectos reproducidos con `fixtures/legacy-es`

| # | Caso | Resultado PoC 1.0 | Esperado | Causa |
|---|---|---|---|---|
| 1 | `@POST @Path("/alta") alta()` tras un `@GET` | `GET /alta` | `POST /expedientes/alta` | Ventana fija de 10 líneas hacia atrás; `elseif` evalúa GET primero |
| 2 | `ayudante()` sin anotaciones | Reportado como `GET /{id}` | No es endpoint | Misma ventana: hereda anotaciones del método anterior |
| 3 | Javadoc "This class exposes…" antes de `@Path("/expedientes")` | Rutas sin prefijo `/expedientes` | `/expedientes/{id}` | `classDecl` encuentra "class exposes" en el comentario |
| 4 | Paquete `...transaction.servicio` | Capa `Entrada / Web` | `Negocio` | `-match 'action'` por subcadena dentro de "transaction" |
| 5 | Paquete `...negocio` | Capa `Otros` | `Negocio` | Diccionario solo en inglés |
| 6 | `public static class Interna` | FQN `org.example.legacy.gestion.negocio.Interna`, contado como top-level | `...GestorNegocio.Interna` | Sin pila de anidamiento; posibles colisiones en `typesByFqn` |
| 7 | `src/test/**` | Incluido; la **única** dependencia interna detectada viene de un test | Excluido por defecto | Sin noción de source set |
| 8 | `import org.example.legacy.gestion.web.*;` + campo `ExpedienteRest` | Sin dependencia | `negocio → web` | Imports comodín ignorados; uso sin import no detectado |
| 9 | `// import javax.jms.Queue;` | Hallazgos `JMS` + `javax` | Ninguno | Reglas sobre texto con comentarios |
| 10 | `/* ... new URL( ... */` | Hallazgo `HTTP cliente` | Ninguno | Ídem |
| 11 | `<dependencyManagement>` | Listado como dependencia, versión `${v}` | `managed=true`, sin arista | Regex `<dependency>` global |
| 12 | `pom.xml` con `<parent>` | No se extrae GAV propio ni parent | GAV heredado | No implementado — **bloquea el enlace entre repos** |
| 13 | `jboss-web.xml` `context-root` | No se combina con rutas REST | Ruta completa | No implementado |
| 14 | 6 imports `javax.*` en un fichero | 6 filas de riesgo | 1 fila, `count=6` | Riesgo por línea |
| 15 | `@Entity` en paquete `modelo` | Capa `Persistencia` | `Modelo` (estereotipo Entity) | Anotación del fichero decide la capa |

## Observaciones de diseño
- **Monolito de 444 líneas**: extracción, modelo y render acoplados; imposible sustituir el
  extractor o añadir vista de portfolio sin reescribir.
- **Reglas como código**: 20 regex inline sin id, versión ni tests.
- **Rendimiento**: `Get-ChildItem -Recurse` recorre `node_modules`/`target` y filtra después;
  `Count-Line` es O(tamaño) por cada match (cuadrático en ficheros grandes); 20 `-match` por línea.
- **Codificación**: `ReadAllText` sin BOM asume UTF-8; los fuentes ISO-8859-1 habituales en
  código legacy en español se degradan silenciosamente.
- **XML por regex**: frágil ante atributos, namespaces, CDATA y descriptores en varias líneas.
- **Validación efímera**: el estado confirmado/descartado no se puede persistir entre ejecuciones.
- **Sin vista multi-repo**: la necesidad declarada no está cubierta por diseño.
