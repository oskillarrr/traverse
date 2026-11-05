## ADDED Requirements

### Requirement: Documentación arc42 en Markdown
El sistema SHALL generar por sistema un `arc42.md` con las 12 secciones de arc42, prerrellenadas con
evidencias y marcando con ✍️ lo que requiere conocimiento humano. Incluye un catálogo de
integraciones con dirección, tipo, contraparte, clave, modo, componente, evidencia y confianza.

#### Scenario: Integración saliente con contraparte
- **WHEN** el sistema A llama por HTTP a un endpoint expuesto por B
- **THEN** el catálogo de A contiene una fila saliente, de tipo http, contraparte B, modo síncrono y la evidencia `fichero:línea` de la llamada en A

### Requirement: Documento de portfolio
El sistema SHALL generar `portfolio.md` con catálogo de sistemas, mapa, matriz de integraciones
sistema × sistema, catálogo de integraciones entre sistemas, datos compartidos, librerías y
dependencias externas.

#### Scenario: Matriz
- **WHEN** A consume de B por JMS y por librería
- **THEN** la celda (A, B) de la matriz indica `jms, library`

### Requirement: Relaciones de confianza baja
Las relaciones de confianza baja SHALL dibujarse atenuadas y con la etiqueta "confianza baja" en todos
los formatos de diagrama, y `--min-confidence` SHALL excluirlas de los diagramas sin quitarlas de
tablas ni CSV.

#### Scenario: Filtro
- **WHEN** se ejecuta con `--min-confidence medium`
- **THEN** `landscape.svg` no contiene relaciones de confianza baja y `links.csv` sí las contiene

### Requirement: Salidas reproducibles
Dos ejecuciones sobre la misma entrada SHALL producir ficheros idénticos byte a byte (salvo con
`SOURCE_DATE_EPOCH` sin fijar, solo la marca temporal).

#### Scenario: Doble ejecución
- **WHEN** se genera el informe dos veces con la misma entrada y la misma marca temporal
- **THEN** todos los ficheros son idénticos
