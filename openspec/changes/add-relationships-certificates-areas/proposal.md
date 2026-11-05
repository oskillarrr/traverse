# Change: relaciones por sistema, certificados, vistas por área y muestreo de validación (v0.8.2)

## Why
Tras la primera ejecución completa en un entorno real, el equipo de arquitectura pidió saber, por aplicación,
**a quién llama y quién le llama**. También pidió una forma de medir la fiabilidad del análisis antes
de presentarlo.

En la ejecución apareció además una relación con mTLS que no se enlazaba:
- un portal llama a `/sign-gateway/signService`;
- la ruta de la pasarela de firma se deduce del nombre del artefacto Maven, con la versión incluida
  (`/sign-gateway-2.3`).

## What Changes
- **Sección "Relaciones con otras aplicaciones"** en cada informe (HTML y report.md), con resumen
  ejecutivo:
  - **A quién llama**: salientes agrupadas por aplicación, con protocolos, endpoints, transporte,
    autenticación y entornos.
  - **Quién le llama**: sistemas analizados que le llaman y datos compartidos.
  - **Interfaces sin consumidor identificado**, separadas en API, pantallas y otras.
  - **Consumidores declarados en el lado proveedor**: los certificados de aplicaciones o servidores de
    confianza.
- **Certificados públicos**:
  - Se leen ficheros PEM/DER (también sin extensión, como guardan algunas pasarelas las aplicaciones registradas), las entradas
    de truststores JKS y la cadena de las entradas con clave.
  - De cada uno se obtiene sujeto, emisor, caducidad, si es CA y si es la identidad propia.
  - Si es de persona física no se guarda el sujeto. Los CN con forma de host se cifran como el resto
    de hosts.
  - Los certificados caducados o que caducan en menos de 90 días generan un hallazgo.
- **Enlace por context root versionado**: una llamada a `/app/x` enlaza con un sistema que expone
  `/app-1.2/x` cuando la ruta se dedujo del nombre del artefacto Maven.
- **Vistas por área**: si el manifiesto tiene más de un grupo, se generan `areas/<grupo>/`
  (`index.html`, `portfolio.md`, diagramas) a partir del portfolio global. Incluyen los sistemas
  del área y los de otras áreas relacionados con ella.
- **`validation-sample.csv`**: muestra determinista y estratificada de 20 afirmaciones (relaciones,
  llamadas salientes, autenticación, riesgos altos y estructura), con columna de veredicto, para
  medir la precisión.

## Impact
- Campo nuevo `certificates` en los hechos (aditivo, esquema 1.5.0).
- Salidas nuevas: `areas/`, `validation-sample.csv` y la sección de relaciones.
- Sin red, sin ejecución, sin descifrar: solo se leen datos públicos de los certificados.
