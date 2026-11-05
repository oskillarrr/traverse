# Change: certificados y contratos con varios proveedores (v0.9.1)

## Why
La primera ejecución de 0.9.0 en un entorno real mostró tres cosas:
- **Autoridades marcadas como personas.** 15 certificados de CA se marcaron como de persona física y
  perdieron el sujeto. Algunas CA llevan correo o un CIF en el sujeto.
- **Host en claro.** El nombre de un servidor aparecía en claro dentro de una OU
  (`OU=srv.dominio.dom`). Solo se cifraba cuando estaba en el CN.
- **Destino que no decide el código.** Una interfaz de aviso (callback) la implementaban dos
  sistemas analizados. El consumidor se enlazaba con ambos con confianza alta, aunque a cuál
  llama lo decide la configuración en ejecución.

## What Changes
- **Una CA nunca es persona física.**
- **Los nombres de host dentro de cualquier atributo del sujeto o del alias** de certificados que
  no son CA se sustituyen por `‹host›` y se cifran como el resto de hosts.
- **Relaciones con varios candidatos.** Cuando varios sistemas implementan la interfaz llamada:
  - la relación baja a confianza media;
  - lleva el campo `candidates`;
  - los informes indican "N sistemas implementan la interfaz";
  - `links.csv` añade la columna `candidateProviders` al final.

## Impact
- Specs: repo-extraction, portfolio-aggregation.
- Código: `internal/scan/certs.go`, `internal/portfolio/aggregate.go`, `internal/model/portfolio.go`
  y `internal/report`.
