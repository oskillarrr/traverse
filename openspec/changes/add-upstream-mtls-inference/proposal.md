# Change: mTLS entrante terminado en un intermediario (v0.9.4)

## Why
En la revisión del análisis se señaló un hueco: cuando un intermediario (un proxy, un API gateway, un
appliance de seguridad o un balanceador) termina el TLS mutuo, la aplicación solo recibe una
petición HTTP. El certificado del cliente, o los datos que necesita de él, llegan en una cabecera.
Ese mTLS no aparece en ningún descriptor del servidor, así que Traverse no lo veía.

La fuente fiable de quién exige el mTLS (un consumidor, una norma) es externa al código y no se
importa. Lo que sí está en el código es una señal sólida: las API que leen un certificado de una
cabecera tienen detrás una conexión mTLS.

## What Changes
- **Reglas `TLS-UPSTREAM-*`**:
  - certificado leído de una cabecera, con confianza media;
  - datos del certificado en cabeceras (sujeto, serie, verificación), con confianza baja;
  - cabecera de certificado configurada, con confianza baja.
- **Marcado por endpoint**: las interfaces entrantes del fichero que lee la cabecera, o todas las API
  del módulo si se lee en un filtro o interceptor, pasan a:
  - transporte `mtls-upstream`;
  - autenticación `client-cert (header)`;
  - rol `api`.
- **Estado del mTLS entrante** por sistema: lo exige el servidor, inferido en un intermediario, o no
  visible. Se muestra en el informe, `report.md`, el resumen ejecutivo y `arc42.md`.
- **`arc42.md`** añade un apartado ✍️ para los requisitos de transporte impuestos por terceros.
- **`contracts.csv`** añade `transport`, `auth`, `role` y `counterpart`.

## Impact
- Specs: repo-extraction, architecture-reporting.
- Código: catálogo de reglas, `internal/scan/classify.go` y `internal/report`.
- Se descarta leer la configuración del intermediario: no está disponible.
