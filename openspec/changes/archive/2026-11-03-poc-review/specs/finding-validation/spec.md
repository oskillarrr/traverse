## ADDED Requirements

### Requirement: Huella estable de hallazgos
Cada hallazgo SHALL tener un identificador derivado de `ruleId`, repositorio, fichero y ancla
estructural (FQN o XPath), independiente del número de línea.

#### Scenario: Edición no relacionada
- **WHEN** se añaden 20 líneas al principio de un fichero y se re-ejecuta
- **THEN** los hallazgos existentes conservan su identificador

### Requirement: Validaciones persistentes
El sistema SHALL leer un fichero `validaciones.csv` (`id, estado, autor, fecha, nota`) mantenido
por el arquitecto y SHALL aplicar su estado a los hallazgos en cada re-ejecución.

#### Scenario: Falso positivo descartado
- **WHEN** un hallazgo tiene estado `descartado` en `validaciones.csv`
- **THEN** no aparece en riesgos ni diagramas, y sí en el anexo "Descartados" con la nota

#### Scenario: Validación huérfana
- **WHEN** un id de `validaciones.csv` ya no corresponde a ningún hallazgo
- **THEN** se lista como "validación huérfana" para revisión

### Requirement: Diferencias entre ejecuciones
El sistema SHALL comparar dos `facts.json` del mismo repositorio y SHALL informar componentes,
dependencias, contratos y riesgos añadidos y eliminados.

#### Scenario: Nueva dependencia externa
- **WHEN** entre dos ejecuciones aparece un contrato consumido nuevo
- **THEN** figura en `diff.html` en la sección "Añadido"

### Requirement: Medición de calidad del analizador
El sistema SHALL calcular precisión por regla a partir de las validaciones (`confirmado` vs
`descartado`) para guiar el ajuste del catálogo de reglas.

#### Scenario: Regla ruidosa
- **WHEN** una regla tiene precisión < 60 % con ≥ 10 hallazgos validados
- **THEN** se señala en el informe de calidad de reglas
