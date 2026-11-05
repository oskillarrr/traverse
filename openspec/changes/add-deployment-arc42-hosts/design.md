# Design

## Lector YAML (internal/yamlite)
Subconjunto de YAML 1.2 usado por Compose, Kubernetes y Helm: mapeos y secuencias de bloque,
escalares planos y entrecomillados, escalares de bloque (`|`, `>`), colecciones flow en una línea
(o equilibradas en varias), comentarios y multidocumento. Los anclajes y etiquetas se ignoran.
Nunca ejecuta nada y nunca falla del todo: ante una entrada mal formada devuelve el árbol parcial
y un error que se registra como diagnóstico `yaml-partial`. En las plantillas Helm, las directivas
`{{ }}` se neutralizan conservando los números de línea.

**Alternativa descartada:** una librería YAML de terceros. Rompería la regla de cero dependencias
y la decisión D3 (sin herramientas o librerías que haya que homologar).

## Cifrado de hosts (internal/secure)
- Clave maestra de 32 bytes: fichero (`traverse keygen`) o frase de paso con PBKDF2-SHA256 y 600k
  iteraciones. La sal es fija para que los pseudónimos sean estables entre ejecuciones y equipos.
- HKDF-SHA256 deriva dos subclaves independientes: pseudónimo y cifrado.
- Pseudónimo: `host-` + 12 hex de HMAC-SHA256(host en minúsculas).
- Bóveda: AES-256-GCM, con el token como datos asociados. El nonce se deriva del host por HMAC
  (tipo SIV), lo que da salidas reproducibles y no revela más de lo que ya revela el token.
- Punto de salida único: `scan.sealHosts` reemplaza o descarta todos los hosts antes de cualquier
  escritura. Los extractores solo dejan el host en memoria.

## Documentación arc42
- El catálogo de integraciones combina las relaciones del portfolio (con contraparte) y los
  contratos propios sin contraparte (expuestos sin consumidor identificado, llamadas externas con
  host cifrado, recursos del servidor).
- La evidencia del consumidor se localiza con el mismo algoritmo que crea las relaciones
  (`portfolio.MatchScore`).
- Los escenarios de ejecución recorren en anchura las dependencias de código (profundidad 6)
  desde endpoints, suscripciones y jobs batch hasta los contratos de salida.
