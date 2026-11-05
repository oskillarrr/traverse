## ADDED Requirements

### Requirement: Privacidad del sujeto de certificados
Un certificado de autoridad (CA) SHALL NOT considerarse de persona física. En los certificados que
no son CA, todo nombre de host presente en cualquier atributo del sujeto o en el alias SHALL
sustituirse por un marcador y SHALL cifrarse como el resto de hosts.

#### Scenario: Host en una OU
- **WHEN** un certificado de servidor tiene `OU=srv01.corp.example.net`
- **THEN** el sujeto muestra `OU=‹host›` y el host se guarda como seudónimo

#### Scenario: CA con número en el CN
- **WHEN** una CA tiene un CN con un número de 8 dígitos
- **THEN** no se marca como persona física y conserva el sujeto

### Requirement: Exclusiones en reglas
Una regla SHALL poder declarar `exclude`, una expresión regular. Una coincidencia cuyo texto también
encaje con `exclude` SHALL descartarse. Las reglas de credenciales SHALL exigir que la palabra clave
sea un segmento completo de la clave y SHALL descartar los valores que son una URL o una ruta.

#### Scenario: Palabra que contiene la clave
- **WHEN** un fichero de configuración tiene `job.secretario.label=Secretario`
- **THEN** no se registra ninguna credencial

#### Scenario: URL como valor
- **WHEN** un fichero de configuración tiene `url.pwd.error=https://example.org/error`
- **THEN** no se registra ninguna credencial

### Requirement: Credenciales en cadenas de conexión
El sistema SHALL marcar como riesgo alto un usuario y una contraseña incluidos en una cadena de
conexión (`esquema://usuario:clave@` o `jdbc:oracle:thin:usuario/clave@`), sin copiar el valor al
informe. Las variables (`${...}`) SHALL NOT marcarse.

#### Scenario: Cadena JDBC de Oracle con credenciales
- **WHEN** una propiedad vale `jdbc:oracle:thin:APP_USER/s3cr3t@//db.example.org:1521/SVC`
- **THEN** se registra un hallazgo `SEC-SECRET-URL` de severidad alta, solo con fichero y línea
