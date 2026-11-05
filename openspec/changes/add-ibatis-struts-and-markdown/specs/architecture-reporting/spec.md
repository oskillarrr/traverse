## ADDED Requirements

### Requirement: Resumen ejecutivo
Los informes HTML (por sistema y de portfolio), `report.md` y `portfolio.md` SHALL incluir un
resumen ejecutivo con tamaño y stack, despliegue, integraciones, vulnerabilidades (si se evaluaron),
riesgos y acoplamiento.

#### Scenario: Resumen por sistema
- **WHEN** se genera el informe de un sistema
- **THEN** `report.html` y `report.md` incluyen una sección "Resumen ejecutivo" con esas cifras

### Requirement: Exportación del análisis a Markdown
El sistema SHALL generar, por cada sistema, un `report.md` con el análisis completo y poblado
(métricas, lenguajes, tecnologías, integraciones, acceso a datos, despliegue, vulnerabilidades,
riesgos y acoplamiento), sin secciones pendientes de completar, equivalente al `report.html`.

#### Scenario: report.md generado
- **WHEN** se ejecuta un análisis
- **THEN** cada carpeta de sistema contiene `report.md`, y el resultado es byte a byte reproducible entre ejecuciones con el mismo `SOURCE_DATE_EPOCH`
