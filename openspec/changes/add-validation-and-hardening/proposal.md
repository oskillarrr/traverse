# Change: validaciones persistentes, diff entre ejecuciones y endurecimiento para el piloto

## Why
La v0.1.0 cubre extracción, portfolio e informes. Para usarla en un assessment real con muchas
aplicaciones faltan tres cosas: que la validación humana de hallazgos no se pierda al volver a
ejecutar, poder comparar ejecuciones (qué ha cambiado) y validar la herramienta en Windows sobre
código real antes de difundirla a toda la comunidad.

## What Changes
- `traverse scan --validations validaciones.csv`: aplica estados (`confirmado`, `probable`, `descartado`,
  `no-verificable`) por id estable; descartados fuera de riesgos y diagramas; anexo de huérfanas.
- `traverse diff OLD_DIR NEW_DIR`: componentes, contratos, enlaces y riesgos añadidos y eliminados.
- Informe de precisión por regla a partir de las validaciones.
- Matriz de capas permitidas configurable en `traverse.json`.
- Piloto: 3 repositorios relacionados (librería común + 2 consumidores), Windows 10/11 con PS 5.1,
  medición de precisión/recall por categoría y tiempos.

## Impact
- Specs: `finding-validation` (ADDED), `repo-extraction` (ADDED: matriz de capas configurable).
- Código: `cmd/traverse` (subcomando `diff`, opción `--validations`), `internal/report`, `internal/scan/layers.go`.
- Sin cambio incompatible en `facts.json` (campo opcional `validation` → schema 1.1.0).
