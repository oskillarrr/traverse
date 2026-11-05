# Design

## Señales (estáticas)
- **Autenticación**: imports y anotaciones de los frameworks (Spring Security, OAuth2/JWT, SAML,
  CAS, Keycloak, Kerberos, LDAP), interceptores WS-Security (WSS4J), cabeceras de API key y
  Basic/Bearer en clientes, y `<login-config>` de web.xml.
- **mTLS**: `auth-method CLIENT-CERT` y `transport-guarantee CONFIDENTIAL` (web.xml); `clientAuth`/
  `certificateVerification` (Tomcat); `verify-client` y `security-realm` con keystore+truststore
  (WildFly); `server.ssl.client-auth=need|want` (Spring Boot); `KeyManagerFactory`/`loadKeyMaterial`/
  `TLSClientParameters`/`javax.net.ssl.keyStore` (código).
- **Transporte**: esquema de la URL (https→tls, http→plain) en literales de código, direcciones WSDL
  y clientes CXF. Se almacena en `Contract.transport`.
- **Firma electrónica**: se detecta por estándares (XML-DSig, CMS, PDF, AdES, RFC 3161, PKCS#11).
  Las plataformas propias de un país quedan en paquetes de reglas opcionales (make-core-agnostic).

## Representación
- Los mecanismos son hallazgos de categoría Authentication / Transport Security (ya exportables a
  SARIF). La vista de seguridad del informe los agrupa por tecnología.
- El inventario de terceros se deriva del catálogo de integraciones: contrapartes externas o con host,
  con su transporte y auth.

## Límites y alternativas descartadas
- No se infiere auth por endpoint salvo lo evidente (Basic/Bearer en el propio cliente); el mapeo
  fino regla→endpoint daría falsos positivos.
- No se consulta red ni se ejecuta nada. El análisis ve configuración, no cumplimiento en runtime: se
  dice explícitamente en el informe.
