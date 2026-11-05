## ADDED Requirements

### Requirement: A quién llama y quién le llama
Cada informe de sistema SHALL incluir una sección de relaciones con:
- las aplicaciones a las que llama, agrupadas por contraparte;
- los sistemas analizados que le llaman;
- los sistemas con los que comparte datos;
- las interfaces sin consumidor identificado, por tipo;
- los certificados de aplicaciones o servidores de confianza.

#### Scenario: Interfaces sin consumidor
- **WHEN** un sistema expone 14 rutas de API y ningún sistema analizado las llama
- **THEN** el informe indica "Sin consumidor identificado: 14 API" y explica que el consumidor solo se conoce si también se analiza

### Requirement: Muestra de validación
El portfolio SHALL incluir `validation-sample.csv` con una selección determinista y estratificada de
afirmaciones del análisis. Cada fila SHALL llevar su evidencia y una columna de veredicto para revisión
humana.

#### Scenario: Misma entrada, misma muestra
- **WHEN** se analiza dos veces el mismo código
- **THEN** `validation-sample.csv` contiene las mismas afirmaciones en el mismo orden
