## ADDED Requirements

### Requirement: mTLS terminado en un intermediario
El sistema SHALL detectar el código que lee de una cabecera HTTP el certificado de cliente o datos de
él. SHALL marcar como transporte `mtls-upstream` y autenticación `client-cert (header)`:
- las interfaces entrantes del fichero que lee la cabecera;
- si el fichero es un filtro o interceptor sin interfaces propias, todas las API de su módulo.

La detección SHALL ser inferida y SHALL NOT afirmar quién exige el mTLS ni qué consumidores lo usan.

#### Scenario: Controlador que recibe el certificado
- **WHEN** un método `POST /api/receipts/delivery` declara `@RequestHeader("X-Client-Cert") String pem`
- **THEN** esa interfaz tiene transporte `mtls-upstream` y las demás interfaces del sistema no cambian

#### Scenario: Filtro del módulo
- **WHEN** `ClientCertFilter.java` lee `request.getHeader("SSL_CLIENT_CERT")` en el módulo `web`
- **THEN** todas las API del módulo `web` tienen transporte `mtls-upstream` y las pantallas no cambian

#### Scenario: Acceso a una propiedad
- **WHEN** el código lee `props.get("cert.path")`
- **THEN** no se infiere mTLS
