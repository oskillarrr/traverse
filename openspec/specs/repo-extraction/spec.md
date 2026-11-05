# repo-extraction Specification

## Purpose
Extract facts from one repository (inventory, modules, components, dependencies, contracts, findings) offline, read-only and with evidence.

## Requirements

### Requirement: Extracción de solo lectura y sin red
El extractor SHALL leer únicamente ficheros del repositorio indicado, SHALL escribir solo en la
carpeta de salida (fuera del repositorio) y SHALL NOT abrir conexiones de red, resolver DTD/XSD
externos ni ejecutar procesos.

#### Scenario: XML con DTD externo
- **WHEN** un `web.xml` declara `<!DOCTYPE web-app PUBLIC "..." "http://java.sun.com/dtd/web-app_2_3.dtd">`
- **THEN** el fichero se parsea sin intentar descargar el DTD
- **AND** no se produce ningún acceso de red

#### Scenario: Salida dentro del repositorio
- **WHEN** `--out` está dentro de un repositorio analizado o lo contiene
- **THEN** la ejecución termina con error antes de leer ningún fichero

### Requirement: Comentarios y literales no generan hallazgos
El extractor SHALL eliminar comentarios de línea, de bloque, Javadoc, literales de cadena y
text blocks antes de aplicar reglas de tecnología sobre ficheros Java, preservando los números
de línea originales.

#### Scenario: Import comentado
- **WHEN** un fichero contiene `// import javax.jms.Queue;`
- **THEN** no se registra hallazgo `JMS` ni `Java EE / javax` para esa línea

#### Scenario: Palabra clave en Javadoc
- **WHEN** un Javadoc contiene "This class exposes expedientes" antes de la declaración de clase
- **THEN** la declaración de clase detectada es la real
- **AND** el `@Path` a nivel de clase se conserva como prefijo de ruta

### Requirement: Asociación exacta anotación-declaración
El extractor SHALL asociar cada bloque de anotaciones (incluidas anotaciones multilínea) a la
declaración inmediatamente siguiente y a ninguna otra.

#### Scenario: Método sin anotación tras método anotado
- **WHEN** `ayudante()` sin anotaciones sigue a un método `@GET`
- **THEN** `ayudante()` NO aparece como endpoint

#### Scenario: Verbo de cada método
- **WHEN** una clase tiene `@GET @Path("/{id}") uno()` y después `@POST @Path("/alta") alta()`
- **THEN** se registran exactamente `GET .../{id}` y `POST .../alta`

### Requirement: Identidad correcta de tipos
El extractor SHALL calcular el FQN de clases anidadas como `paquete.Externa.Interna` y SHALL
marcar cada tipo con `sourceSet` (`main`, `test`, `generated`) según su ruta.

#### Scenario: Clase interna
- **WHEN** `GestorNegocio` declara `public static class Interna`
- **THEN** el componente se registra como `...negocio.GestorNegocio.Interna` con `kind=nested-class`

#### Scenario: Código de test
- **WHEN** un tipo reside bajo `src/test/`
- **THEN** se registra con `sourceSet=test`
- **AND** se excluye por defecto de componentes, grafos, métricas y riesgos (salvo reglas de secretos)

### Requirement: Dependencias internas tipadas
El extractor SHALL registrar dependencias entre tipos del repositorio procedentes de imports
explícitos, imports comodín resolubles contra tipos conocidos, referencias en el mismo paquete,
`extends`/`implements`, campos e inyección (`@Inject`, `@EJB`, `@Autowired`), indicando `kind` y
`confidence`.

#### Scenario: Import comodín
- **WHEN** `GestorNegocio` hace `import ...gestion.web.*;` y declara un campo `ExpedienteRest`
- **THEN** existe la dependencia `GestorNegocio → ExpedienteRest` con `kind=wildcard`

### Requirement: Clasificación de capas explicable
El extractor SHALL asignar capa por tipo usando segmentos completos de paquete contra un
diccionario configurable en español e inglés, SHALL permitir overrides por aplicación y SHALL
registrar el motivo.

#### Scenario: Subcadena engañosa
- **WHEN** el paquete es `...gestion.transaction.servicio`
- **THEN** la capa es `Business` con `layerReason=package:servicio`
- **AND** no es `Presentation`

### Requirement: Identidad y dependencias Maven
El extractor SHALL extraer el GAV de cada `pom.xml` heredando `groupId`/`version` del `parent`,
SHALL resolver propiedades disponibles en el reactor y SHALL separar `dependencyManagement` y
dependencias de plugins de las dependencias efectivas.

#### Scenario: dependencyManagement
- **WHEN** una dependencia solo aparece en `<dependencyManagement>`
- **THEN** se registra con `managed=true` y no genera arista de dependencia

### Requirement: Unidades de despliegue y rutas completas
El extractor SHALL combinar `context-root` (jboss-web.xml / application.xml), `@ApplicationPath`
o `servlet-mapping`, `@Path` de clase y de método para producir la ruta HTTP expuesta completa.

#### Scenario: Ruta completa
- **WHEN** `context-root=/pedidos`, `@ApplicationPath("/api")`, clase `@Path("/pedidos")`, método `@GET @Path("/estado")`
- **THEN** el contrato provisto es `GET /pedidos/api/pedidos/estado`

#### Scenario: Context root desde EAR
- **WHEN** `application.xml` declara `<web-uri>case-web.war</web-uri><context-root>/cases</context-root>`
- **THEN** los endpoints del módulo `case-web` llevan el prefijo `/cases`

### Requirement: Codificación de ficheros legacy
El extractor SHALL decodificar UTF-8 (con o sin BOM) y UTF-16 con BOM y, si el contenido no es UTF-8 válido,
como Windows-1252, registrando la codificación en el inventario de ficheros.

#### Scenario: Fuente ISO-8859-1
- **WHEN** un `.java` contiene `Gestión` codificado en ISO-8859-1
- **THEN** los identificadores se leen sin caracteres de reemplazo y `files.csv` indica `windows-1252`

### Requirement: Inventario multitecnología
El extractor SHALL inventariar todos los ficheros con lenguaje, categoría, líneas de código, comentario y en blanco para cualquier tecnología registrada, y SHALL registrar las librerías binarias incluidas en el repositorio como dependencias `vendored-jar`.

#### Scenario: Repositorio legacy sin gestor de dependencias
- **WHEN** existe `WebContent/WEB-INF/lib/commons-lang-2.6.jar`
- **THEN** se registra la dependencia `jar:commons-lang` versión `2.6`

### Requirement: Layout estándar de Subversion
El extractor SHALL analizar solo `trunk` cuando un directorio contiene `trunk`, `branches` y `tags`, salvo que se indique `--svn-all`, y SHALL dejar constancia en el informe.

#### Scenario: Checkout de la raíz SVN
- **WHEN** se analiza una carpeta con `trunk/`, `branches/b1/` y `tags/v1/`
- **THEN** no se analiza ningún fichero de `branches/` ni `tags/`
- **AND** `repo.layout` vale `svn-standard (trunk only)`
