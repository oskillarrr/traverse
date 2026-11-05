## ADDED Requirements

### Requirement: Vista de autenticación e inventario de terceros
Los informes SHALL incluir una sección de autenticación y seguridad de transporte (mecanismos
detectados, mTLS, puntos débiles) y un inventario de sistemas de terceros (contraparte, dirección,
protocolo, transporte, autenticación y evidencia), además de columnas de transporte/auth en el
catálogo de integraciones y una línea de seguridad en el resumen ejecutivo.

#### Scenario: Sección de seguridad
- **WHEN** se detectan mecanismos de autenticación
- **THEN** `report.html` y `report.md` incluyen la sección "Autenticación y seguridad de transporte"

#### Scenario: Punto débil por transporte
- **WHEN** hay una llamada saliente con `transport = plain`
- **THEN** aparece como punto débil en la vista de seguridad
