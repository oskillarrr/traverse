## ADDED Requirements

### Requirement: Matriz de capas configurable
El extractor SHALL permitir definir en `traverse.json` qué dependencias entre capas están prohibidas,
sustituyendo la matriz por defecto.

#### Scenario: Arquitectura hexagonal
- **WHEN** `layers.forbidden` declara `{"Model": ["Persistence", "Integration", "Presentation"]}`
- **THEN** solo esas dependencias se informan como violaciones de capas
