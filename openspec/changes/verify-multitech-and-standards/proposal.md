# Change: verificación multitecnología y formatos estándar (v0.2.0)

## Why
Antes de la validación por parte de los arquitectos había que demostrar con evidencias tres cosas:

1. Que Traverse analiza todo tipo de proyectos.
2. Que extrae las piezas que necesita un arquitecto.
3. Que sus salidas siguen formatos y convenciones oficiales.

La ejecución sobre 13 proyectos reales mostró dos problemas:

- Siete tecnologías se quedaban en inventario: .NET, Kotlin, JS/TS, Python, PHP, Go y mainframe.
- Había 15 defectos, entre ellos un DSL inválido detectado por la herramienta oficial de Structurizr.

Ver `evidence/VERIFICACION-2026-11-03.md`.

## What Changes
- Análisis estructural de Kotlin y C# reutilizando el parser de Java (dialectos).
- Extractor ligero para JS/TS, Python, PHP, Go y Ruby: módulos, imports, rutas y tablas ORM.
- Extractor mainframe: COBOL, CICS, JCL, BMS, CSD y HLASM.
- Páginas de servidor como endpoints (JSP, ASP.NET, Razor Pages, ASP clásico).
- Exportaciones ArchiMate 3.1 (The Open Group), SARIF 2.1.0 (OASIS) y CycloneDX 1.6 (ECMA-424).
- `scripts/validate-outputs.sh` para validar todas las salidas con herramientas y esquemas oficiales.
- Corrección de los 15 defectos del informe de verificación.

## Decisions to discuss (bloquean v0.3)
- **D1 Convenciones oficiales.** Implementados C4, ArchiMate, SARIF, CycloneDX y arc42 como referencia de contenido. ¿Hay una metodología o plantilla obligatoria en la organización (Métrica v3, plantilla corporativa, ENS)?
- **D2 Documento generado.** ¿arc42 prerrellenado (Markdown o AsciiDoc), plantilla corporativa o ninguno?
- **D3 Vista de despliegue.** ¿Modelar Dockerfile, Compose y Kubernetes? Requiere un lector YAML propio para no añadir dependencias.
- **D4 Vulnerabilidades.** ¿Cruce offline del SBOM con OSV/NVD o delegar en Dependency-Track?
- **D5 Hosts.** ¿Activar `--include-hosts` por defecto?
- **D6 Enlaces de baja confianza.** ¿Mostrar atenuados, ocultar o configurar?

## Impact
- Specs: `repo-extraction`, `portfolio-aggregation`, `architecture-reporting` (ADDED).
- Esquema de hechos: campos aditivos `module.artifact`, `deployable.contextRootSource`, `sharedTable.basis` (schema 1.1.0).
- Sin cambios incompatibles en la CLI.
