# finding-validation Specification

## Purpose
Make findings traceable and stable across runs so architects can validate them.

## Requirements

### Requirement: Huella estable de hallazgos
Cada hallazgo SHALL tener un identificador derivado de `ruleId`, repositorio, fichero y ancla
estructural (FQN o XPath), independiente del número de línea.

#### Scenario: Edición no relacionada
- **WHEN** se añaden 20 líneas al principio de un fichero y se re-ejecuta
- **THEN** los hallazgos existentes conservan su identificador

### Requirement: Columna de validación
`findings.csv` SHALL incluir una columna `validation` vacía para que el arquitecto registre su decisión.

#### Scenario: Exportación
- **WHEN** se genera `findings.csv`
- **THEN** su cabecera incluye `id` y `validation`
