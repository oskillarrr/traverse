# Change: Traverse v2 — pipeline extracción → hechos → agregación de portfolio → informes

## Why
La PoC 1.0 cumple su objetivo de demostrar viabilidad (offline, solo lectura, trazable), pero
al ejecutarla sobre casos realistas (ver `evidence/REVISION-PoC-v1.md`) aparecen tres
limitaciones estructurales que no se resuelven añadiendo más regex al script:

1. **Fidelidad**: el análisis línea a línea sin conciencia de comentarios, anidamiento ni
   bloques de anotación produce errores de bulto (verbos HTTP equivocados, rutas sin prefijo
   de clase, endpoints fantasma, capas mal asignadas, tests mezclados con producción).
2. **Alcance**: la necesidad real es un análisis **multi-repositorio**. La PoC no extrae la
   identidad Maven del propio repo (GAV, parent, módulos), ni contratos expuestos/consumidos
   enlazables entre repos (endpoints, colas, JNDI, tablas). Sin eso no hay vista de portfolio.
3. **Mantenibilidad**: extracción, modelo y presentación están acoplados en un único script;
   las reglas son código; no hay tests; la validación manual de hallazgos se pierde en cada
   re-ejecución.

## What Changes
- **Arquitectura en pipeline** con contrato estable entre etapas:
  `extract` (por repo) → `facts.json` versionado → `aggregate` (portfolio) → `render`.
  El extractor es sustituible (PowerShell hoy; Java/JavaParser o bytecode si se aprueba) sin
  tocar agregación ni informes.
- **Extracción de mayor fidelidad sin dependencias**: pre-pase léxico Java (elimina
  comentarios y literales preservando offsets), asociación anotación→declaración, clases
  anidadas, índice de líneas, parseo XML real con `System.Xml` (DTD deshabilitado),
  separación `main`/`test`, decodificación con fallback Windows-1252.
- **Modelo de hechos**: componentes, dependencias tipadas, módulos Maven, unidades de
  despliegue (WAR/EAR/context-root), contratos **expuestos** y **consumidos** (HTTP, SOAP, JMS,
  JNDI, tablas BD), hallazgos con `ruleId` y huella estable.
- **Agregación de portfolio**: grafo entre repos por GAV, matching de contratos
  productor↔consumidor, matriz de propiedad de datos, ciclos y métricas (Ca/Ce/inestabilidad).
- **Reglas como datos** (`rules/*.psd1`) con id, versión, ámbito de fichero, confianza, riesgo y
  ejemplos positivos/negativos testeados.
- **Clasificación de capas configurable** por tokens de paquete (ES/EN) y overrides por app,
  con motivo de clasificación.
- **Informes**: por app y de portfolio; C4 en Structurizr DSL y PlantUML (texto, offline)
  además de Mermaid/SVG; riesgos agregados por regla/fichero, no por línea.
- **Validación persistente**: fichero sidecar de decisiones (`validaciones.csv`) indexado por
  huella del hallazgo, que sobrevive a re-ejecuciones; diff entre ejecuciones.
- **Calidad**: tests Pester (v3.4 incluida en Windows 10/Server 2016+) con fixtures y
  golden files; matriz PS 5.1 / PS 7.
- **BREAKING**: cambia la estructura de la carpeta de salida y los nombres de CSV. Se mantiene
  un modo `-CompatV1` durante una versión.

## Impact
- Specs afectadas (nuevas): `repo-extraction`, `portfolio-aggregation`,
  `architecture-reporting`, `finding-validation`.
- Código: `ejecutar.ps1` se divide en módulo `Traverse.psm1` + `Invoke-OatExtract.ps1`,
  `Invoke-OatAggregate.ps1`, `Invoke-OatReport.ps1`; `ejecutar.ps1` queda como fachada.
- Operación: nuevo manifiesto `repos.csv` para lotes multi-repo.
- Seguridad: sin cambios en la superficie (sigue sin red ni ejecución); se añade protección
  XXE explícita y marca de clasificación en todas las salidas.
