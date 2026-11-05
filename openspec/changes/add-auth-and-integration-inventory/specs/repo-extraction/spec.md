## ADDED Requirements

### Requirement: Detección de autenticación y mTLS
El sistema SHALL detectar mecanismos de autenticación (contenedor, Spring Security, OAuth2/JWT, SAML,
CAS, Keycloak, Kerberos, LDAP, WS-Security, API key, Basic/Bearer) y de TLS mutuo (CLIENT-CERT,
clientAuth/verify-client, keystore de cliente), como hallazgos de categoría Authentication o
Transport Security. (La firma electrónica se detecta por estándares; ver el cambio make-core-agnostic.)

#### Scenario: Cliente de certificado
- **WHEN** un `web.xml` declara `<auth-method>CLIENT-CERT</auth-method>`
- **THEN** se registra un hallazgo de X.509 client auth (mTLS)

#### Scenario: Cliente OAuth2
- **WHEN** el código importa `org.springframework.security.oauth2...`
- **THEN** se registra un hallazgo OAuth2/OIDC de tipo login de usuarios

### Requirement: Transporte por integración
Cada contrato HTTP/SOAP SHALL indicar su transporte (plain/tls/mtls) cuando sea deducible del esquema
de la URL.

#### Scenario: Llamada saliente sin cifrar
- **WHEN** un cliente llama a `http://host/servicio`
- **THEN** el contrato saliente tiene `transport = plain`
