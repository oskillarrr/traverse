# Change: despliegue, documentación arc42 y hosts cifrados (v0.3.0)

## Why
Aplica las decisiones tomadas tras la verificación v0.2.0 (`verify-multitech-and-standards`).

## Decisions (registro)
| # | Pregunta | Decisión | Fecha |
|---|---|---|---|
| D1 | ¿Formato o metodología obligatoria? | No hay ninguna: se usan estándares internacionales (C4, ArchiMate, SARIF, CycloneDX, arc42) | 2026-11-03 |
| D2 | ¿Documento generado? | arc42 en **Markdown**, con especial cuidado en las **integraciones** | 2026-11-03 |
| D3 | ¿Vista de despliegue? | Sí. Gratuita y sin instalar herramientas: lector YAML propio en Go estándar | 2026-11-03 |
| D4 | ¿Vulnerabilidades? | **Opción B**: cruce sin conexión con un volcado OSV descargado manualmente; Dependency-Track queda como consumidor opcional del SBOM. Se implementa en `add-offline-vulnerability-matching` | 2026-11-03 |
| D5 | ¿Nombres de host? | **Cifrados**: pseudónimo HMAC en informes y valor real en una bóveda AES-256-GCM | 2026-11-03 |
| D6 | ¿Relaciones de confianza baja? | **Confirmado**: atenuadas y etiquetadas por defecto; `--min-confidence medium` las quita de los diagramas y las mantiene en tablas y CSV | 2026-11-03 |

## What Changes
- `arc42.md` por sistema y `portfolio.md` con matriz y catálogo de integraciones.
- Vista de despliegue: Dockerfile, Compose, Kubernetes, OpenShift y Helm, con riesgos de despliegue, entornos de despliegue C4 y nodos ArchiMate.
- Relación de portfolio `deploys` (repositorios de plataforma o GitOps).
- `traverse keygen`, `traverse reveal`, `--key-file`/`TRAVERSE_KEY_FILE`/`TRAVERSE_PASSPHRASE`, `hosts.vault.json`.
- `--min-confidence` y estilo atenuado de las relaciones de confianza baja.
- Robustez: un error de extractor en un fichero se convierte en diagnóstico. Determinismo verificado por test.

## Impact
- Esquema de hechos 1.2.0 (aditivo): `deployment`, `contract.host`, `externalNeed.host`.
- Requiere Go 1.24 para compilar (HKDF y PBKDF2 de la librería estándar).
