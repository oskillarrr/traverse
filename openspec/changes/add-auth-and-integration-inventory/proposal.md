# Change: autenticación, mTLS e inventario de terceros (v0.6.0)

## Why
Revisión de un arquitecto del equipo: para proponer soluciones, lo que más importa es (1) las
**integraciones con aplicaciones de terceros** (entrantes y salientes) y (2) la **autenticación**, en
particular si se usa **mTLS**. El resto suele estar documentado, pero disperso. Traverse cubría las
integraciones a medias y apenas la autenticación. Objetivo: detectar todo lo posible de forma
estática y pensarlo "a lo grande".

## What Changes
- **Reglas de autenticación y transporte** (data-driven, autovalidadas): Spring Security method-level,
  X.509/client-cert, JWT, OAuth2/OIDC, SAML, CAS, Keycloak, Kerberos/SPNEGO, LDAP/AD, WS-Security,
  API key, Basic/Bearer salientes, keystore de cliente (mTLS), contexto TLS propio, verificación TLS
  deshabilitada, protocolo TLS obsoleto, y específicas del sector público español (hoy en un paquete opcional): @firma/AutoFirma, Cl@ve/eIDAS, SIGNGW,
  Valida/DSS, FNMT/CERES, sellado de tiempo (RFC 3161).
- **Descriptores**: `transport-guarantee` y `CLIENT-CERT` en web.xml; conector TLS/clientAuth de
  Tomcat; `https-listener verify-client` y `security-realm` de WildFly; `server.ssl.client-auth` de
  Spring Boot (regla); TLS de clientes CXF.
- **Transporte por integración**: cada contrato HTTP/SOAP lleva `transport` (plain/tls/mtls) inferido
  del esquema de la URL; se marca en rojo lo saliente sin cifrar.
- **Informe**: sección "Autenticación y seguridad de transporte" (mecanismos, mTLS, puntos débiles),
  **inventario de sistemas de terceros** (contraparte, protocolo, transporte, auth, evidencia),
  columnas de transporte/auth en el catálogo de integraciones, y línea de seguridad en el resumen
  ejecutivo. En HTML, report.md y arc42.

## Impact
- Esquema de hechos 1.4.0 (aditivo: `Contract.transport`, `Contract.auth`).
- Más hallazgos de categoría Authentication / Transport Security.
- Sin dependencias nuevas. Sin red. Límite honesto: detecta lo **configurado** en el código; no
  prueba que la infraestructura (balanceador, frontal) **exija** mTLS, ni descubre consumidores
  entrantes no declarados.
