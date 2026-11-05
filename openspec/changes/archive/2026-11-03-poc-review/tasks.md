# Tasks

## 0. Quick wins sobre PoC 1.0 (v1.1, sin cambiar formato de salida)
- [ ] 0.1 Excluir `src/test/**` por defecto (parámetro `-IncluirTests`)
- [ ] 0.2 `Get-Capa`: comparar por segmentos de paquete, no subcadenas; añadir vocabulario ES
- [ ] 0.3 Endpoints: cortar la ventana de anotaciones en la declaración/`}` anterior; no heredar verbo
- [ ] 0.4 Prefijo de clase: buscar `@Path` de clase sobre texto sin comentarios
- [ ] 0.5 Ignorar líneas dentro de comentarios en el bucle de reglas (pre-pase léxico mínimo)
- [ ] 0.6 `pom.xml`: excluir `dependencyManagement` y `plugins` de `08-dependencias-maven`
- [ ] 0.7 Agregar riesgos por (regla, fichero) con recuento
- [ ] 0.8 Sustituir `Count-Line` por índice de offsets
- [ ] 0.9 Incorporar `fixtures/legacy-es` y comprobar resultados esperados

## 1. Estructura y calidad
- [ ] 1.1 Crear módulo `Traverse.psm1` y scripts `Invoke-OatExtract/Aggregate/Report`; `ejecutar.ps1` como fachada
- [ ] 1.2 Tests Pester 3.4-compatibles + golden files; ejecutar en PS 5.1 y PS 7
- [ ] 1.3 Catálogo `rules/*.psd1` con `Positive/Negative` auto-testeados
- [ ] 1.4 Config `traverse.config.psd1` (exclusiones, diccionario de capas, matriz de capas permitida, clasificación)

## 2. Extractor v2
- [ ] 2.1 Recorrido con poda de directorios y lectura UTF-8 estricta → fallback Windows-1252
- [ ] 2.2 Pre-pase léxico Java (comentarios, literales, text blocks) con preservación de líneas
- [ ] 2.3 Declaraciones con pila de llaves (anidadas, enums, records, `@interface`)
- [ ] 2.4 Bloques de anotación multilínea asociados a declaración
- [ ] 2.5 Dependencias tipadas (import, comodín, mismo paquete, extends/implements, campo, inyección)
- [ ] 2.6 Parseo XML seguro (D5) de descriptores; Maven con parent/propiedades/módulos (D6)
- [ ] 2.7 Desplegables y rutas completas; contratos provistos/consumidos (D7)
- [ ] 2.8 Emitir `facts.json` v2.0.0 con orden estable y validación de esquema

## 3. Agregación de portfolio
- [ ] 3.1 Manifiesto `repos.csv` y ejecución por lotes tolerante a fallos
- [ ] 3.2 Grafo Maven entre repos y divergencias de versión
- [ ] 3.3 Matching de contratos con umbral de especificidad
- [ ] 3.4 Matriz de propiedad de datos
- [ ] 3.5 Métricas Ca/Ce/I y ciclos (Tarjan) por paquete/módulo/repo; violaciones de capa

## 4. Informes y validación
- [ ] 4.1 HTML por app + portfolio (índice, landscape, matriz de datos, riesgos priorizados)
- [ ] 4.2 Exportadores Structurizr DSL, C4-PlantUML, Mermaid, SVG
- [ ] 4.3 `validaciones.csv`, huellas estables, anexo de descartados y huérfanas
- [ ] 4.4 `diff` entre ejecuciones; informe de precisión por regla

## 5. Piloto y cierre
- [ ] 5.1 Piloto con 3 repos relacionados autorizados; validar ≥ 10 hallazgos por categoría
- [ ] 5.2 Medir precisión/recall y tiempos en PS 5.1; ajustar reglas
- [ ] 5.3 Resolver Open Questions de design.md (jdeps/JDK, WAR/EAR, standalone.xml, herramienta de diagramas)
- [ ] 5.4 Archivar el change y promover specs a `openspec/specs/`
