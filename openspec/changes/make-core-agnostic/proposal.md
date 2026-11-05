# Change: núcleo agnóstico y paquetes de reglas opcionales (v0.9.0)

## Why
Traverse debe servir para cualquier proyecto del mundo. Hasta 0.8.2 el núcleo llevaba conocimiento
de un único país:
- reglas `GOV-*` del sector público español en el catálogo embebido;
- patrones de DNI/NIE y la redacción "NOMBRE/NIF" de las CA españolas para reconocer certificados
  de persona física;
- una tabla en el código que decidía el tipo de autenticación de cada regla, incluidas las `GOV-*`;
- nombres de proyectos reales en ejemplos, tests y documentación.

## What Changes
- **Catálogo embebido por estándares** (versión 2.0.0, **BREAKING** para quien filtre por id):
  - salen `GOV-AFIRMA`, `GOV-CLAVE`, `GOV-FIRE`, `GOV-VALIDA` y `GOV-FNMT`;
  - `GOV-TIMESTAMP` pasa a `SIGN-TIMESTAMP`;
  - nuevas `SIGN-XMLDSIG`, `SIGN-CMS`, `SIGN-PDF`, `SIGN-ADES`, `SIGN-PKCS11` y `AUTH-EIDAS`.
- **Paquetes de reglas opcionales** en `rulepacks/`, fuera del binario. El primero,
  `es-public-sector.json`, recoge las antiguas `GOV-*` con ids `ES-*`. Se cargan con `--rules` o con
  la clave `"rules"` de `traverse.json`.
- **`authKind` como dato de la regla** (user, service, signature, transport). El motor solo conserva
  el tipo de los hallazgos que él mismo deriva de descriptores estándar (web.xml, server.xml…).
- **Certificados de persona física por estándares**: X.520 (givenName, surname, pseudonym), ETSI EN
  319 412-1 (identificadores semánticos de persona física), correo en un sujeto que no es un host y
  CN con número de identificación.
- **Vocabulario internacional** de entornos y carpetas contenedoras.
- **Ejemplos, tests y documentación con nombres neutros.**

## Impact
- Specs: repo-extraction.
- Código: `internal/rules` (campo `authKind`, test de paquetes), `internal/scan` (classify, certs,
  config), `tools/genrules.py`, `rulepacks/`, `cmd/traverse`.
- Migración: añadir `--rules rulepacks/es-public-sector.json` para conservar las detecciones
  españolas; los ids cambian de `GOV-X` a `ES-X`.
