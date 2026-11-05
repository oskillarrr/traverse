# Tasks

## 1. Validaciones
- [ ] 1.1 Leer `validaciones.csv` (`id,estado,autor,fecha,nota`) y aplicarlo a `findings`
- [ ] 1.2 Excluir `descartado` de riesgos/diagramas; anexo "Descartados" y "Huérfanas"
- [ ] 1.3 Informe de precisión por regla (≥ 10 validados)

## 2. Diff
- [ ] 2.1 `traverse diff` sobre dos carpetas de salida (por `facts.json` / `portfolio.json`)
- [ ] 2.2 `diff.html` y `diff.csv`

## 3. Capas configurables
- [ ] 3.1 `layers.forbidden` en `traverse.json` sustituye la matriz por defecto

## 4. Piloto y homologación
- [ ] 4.1 Ejecutar en Windows 10/11 (amd64) sin privilegios; registrar hash y tiempos
  - [x] 4.1.1 0.4.0 (`1c87fb1c…f1e9`) se ejecuta sin privilegios; `vulndb info` lee el volcado real de Maven (7.226 avisos)
  - [x] 4.1.2 0.4.1 (`bcaad4ab…0635`) queda **suspendida al crearse el proceso**, también con `version` y como administrador. No se puede finalizar ("Access is denied") y no aparece en el historial de protección. Diagnóstico: la seguridad del puesto retiene binarios nuevos sin firmar; no es un fallo de Traverse
  - [ ] 4.1.3 Petición a Seguridad: autorizar por hash o firmar. Como cada versión es un binario nuevo, la solución estable es la 4.3
- [ ] 4.2 3 repos reales relacionados; validar ≥ 10 hallazgos por categoría
- [ ] 4.3 Pipeline en GitLab corporativo con firma de binarios (**prioritaria**: sin ella, cada versión nueva queda bloqueada en los puestos gestionados, como ocurrió con la 4.1.2)
- [ ] 4.4 Ajustar reglas con precisión < 60 %

## 5. Extractores (backlog, cambios separados)
- [ ] 5.1 Extractor opcional de bytecode (WAR/EAR/JAR) para dependencias exactas entre clases
- [ ] 5.2 Componentes y endpoints para C# (ASP.NET MVC/Web API, WCF)
- [ ] 5.3 COBOL: programas, `CALL`, copybooks y transacciones CICS como componentes
- [ ] 5.4 Caché incremental por hash de fichero
