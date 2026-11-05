## ADDED Requirements

### Requirement: Informes por aplicación y de portfolio
El sistema SHALL generar un informe HTML autocontenido por repositorio y un informe de
portfolio con índice navegable, sin recursos remotos y con CSP restrictiva.

#### Scenario: Apertura sin red
- **WHEN** se abre cualquier informe en un equipo sin conectividad
- **THEN** todos los diagramas y tablas se visualizan completos

### Requirement: Modelo C4 en texto
El sistema SHALL exportar el modelo como Structurizr DSL y C4-PlantUML, además de Mermaid:
nivel 1 (landscape del portfolio), nivel 2 (desplegables WAR/EAR, librerías, BBDD, colas) y
nivel 3 (componentes por capa) de cada aplicación.

#### Scenario: Landscape
- **WHEN** se agrega un portfolio de 3 repositorios con relaciones Maven y JMS
- **THEN** `portfolio.dsl` contiene los 3 sistemas, las relaciones etiquetadas por tipo y las dependencias externas

### Requirement: Riesgos agregados y priorizados
El sistema SHALL agregar los hallazgos de riesgo por (`ruleId`, fichero) con recuento y rango de
líneas, y SHALL ofrecer un resumen por aplicación ordenado por severidad × número de ficheros.

#### Scenario: Muchos imports javax en un fichero
- **WHEN** un fichero tiene 6 imports `javax.*`
- **THEN** aparece una única fila de riesgo con `count=6` en lugar de 6 filas

### Requirement: Trazabilidad y reproducibilidad
Cada elemento de informe SHALL enlazar a su evidencia (`repo:file:line`, `ruleId`) y cada
salida SHALL incluir versión de herramienta, de reglas y de esquema; con la misma entrada y
versiones, la salida SHALL ser idéntica salvo la marca temporal.

#### Scenario: Re-ejecución
- **WHEN** se ejecuta dos veces sobre el mismo commit
- **THEN** `facts.json` es byte a byte idéntico excluyendo `scannedAt`

### Requirement: Marca de clasificación
Todas las salidas SHALL incluir una cabecera de clasificación configurable (p. ej. "Uso interno").

#### Scenario: CSV
- **WHEN** se genera cualquier CSV
- **THEN** existe un fichero `LEEME-CLASIFICACION.txt` en la carpeta y la marca aparece en el HTML y en el DSL como comentario
