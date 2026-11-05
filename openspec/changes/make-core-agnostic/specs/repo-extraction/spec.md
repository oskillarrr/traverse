## ADDED Requirements

### Requirement: Catálogo de reglas agnóstico
El catálogo embebido SHALL reconocer solo estándares, frameworks abiertos y productos de uso
general, y SHALL NOT contener reglas propias de un país, sector u organización. La firma
electrónica SHALL detectarse por estándar (XML-DSig, CMS/PKCS#7, firma PDF, AdES, RFC 3161,
PKCS#11).

#### Scenario: Firma XAdES
- **WHEN** el código importa `eu.europa.esig.dss.xades.signature.XAdESService`
- **THEN** se registra un hallazgo `SIGN-ADES` de tipo firma electrónica

#### Scenario: Sin reglas regionales de serie
- **WHEN** se carga el catálogo embebido sin ficheros adicionales
- **THEN** ninguna regla tiene un id de ámbito regional (`GOV-`, `ES-`…)

### Requirement: Paquetes de reglas opcionales
El sistema SHALL cargar paquetes de reglas desde `--rules` y desde la clave `rules` de
`traverse.json`, con rutas relativas al fichero de configuración. El orden de carga SHALL ser:
núcleo, configuración y línea de órdenes. Un paquete del repositorio SHALL NOT sustituir una regla
del núcleo.

#### Scenario: Paquete del sector público español
- **WHEN** se analiza con `--rules rulepacks/es-public-sector.json` un código que importa
  `es.gob.afirma.core.signers.AOSigner`
- **THEN** se registra un hallazgo `ES-AFIRMA` de tipo firma electrónica

#### Scenario: Paquete en la configuración
- **WHEN** `traverse.json` declara `"rules": ["packs/acme.json"]`
- **THEN** se carga `packs/acme.json` relativo a la carpeta del `traverse.json`

### Requirement: Tipo de autenticación como dato
Cada regla de autenticación SHALL poder declarar `authKind` (user, service, signature, transport),
y el hallazgo SHALL heredarlo. Un valor distinto SHALL rechazarse al cargar la regla. Sin
`authKind`, el tipo SHALL derivarse de la categoría.

#### Scenario: Regla propia de login
- **WHEN** un fichero de reglas propio declara una regla con `"authKind": "user"`
- **THEN** sus hallazgos aparecen en "Login de usuarios" sin cambiar el código

### Requirement: Certificados de persona física por estándares
Un certificado SHALL considerarse de persona física, y su sujeto y alias SHALL omitirse, si cumple
alguna de estas condiciones:
- su sujeto tiene givenName, surname o pseudonym;
- su serialNumber lleva un identificador semántico de persona física de ETSI EN 319 412-1;
- su sujeto tiene correo y el CN no es un host;
- su CN contiene un número de identificación.

El sistema SHALL NOT usar formatos de identificación de un país concreto.

#### Scenario: Identificador ETSI de otro país
- **WHEN** un certificado tiene serialNumber `PNOSE-196501011234`
- **THEN** se marca como personal y no se guarda el sujeto

#### Scenario: Sello de entidad
- **WHEN** un certificado tiene serialNumber `VATXX-B12345678` y CN sin número de identificación
- **THEN** no se marca como personal
