## ADDED Requirements

### Requirement: Validaciones persistentes
El sistema SHALL leer un fichero de validaciones (`id, estado, autor, fecha, nota`) mantenido por el
arquitecto y SHALL aplicar su estado a los hallazgos en cada re-ejecución.

#### Scenario: Falso positivo descartado
- **WHEN** un hallazgo tiene estado `descartado` en el fichero de validaciones
- **THEN** no aparece en riesgos ni diagramas, y sí en el anexo "Descartados" con la nota

#### Scenario: Validación huérfana
- **WHEN** un id del fichero de validaciones ya no corresponde a ningún hallazgo
- **THEN** se lista como "validación huérfana" para revisión

### Requirement: Diferencias entre ejecuciones
El sistema SHALL comparar dos carpetas de salida y SHALL informar componentes, contratos, enlaces
y riesgos añadidos y eliminados.

#### Scenario: Nueva dependencia externa
- **WHEN** entre dos ejecuciones aparece un contrato consumido nuevo
- **THEN** figura en `diff.html` en la sección "Añadido"

### Requirement: Precisión por regla
El sistema SHALL calcular la precisión de cada regla a partir de las validaciones.

#### Scenario: Regla ruidosa
- **WHEN** una regla tiene precisión < 60 % con ≥ 10 hallazgos validados
- **THEN** se señala en el informe de calidad de reglas
