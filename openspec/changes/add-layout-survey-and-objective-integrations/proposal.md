# Change: reconocimiento de estructura e integraciones objetivas (v0.8.0)

## Why
Primeras ejecuciones en un entorno real, contrastadas a mano con un inventario de aplicaciones:

- **Una aplicación con copias de versiones**: junto a `trunk/` había una carpeta de desarrollo y otra de
  entregas con decenas de copias. Traverse solo descartaba `branches/` y `tags/`, así que analizó todas
  las copias como una única aplicación: las cifras salieron multiplicadas por más de veinte. El informe
  además decía "trunk only".
- **Una aplicación entregada solo como WAR**: la raíz mezclaba código con BBDD, documentación y
  modelos, y el módulo solo tenía WAR compilados y configuración del servidor. Salieron 0 componentes y
  0 integraciones, aunque las URL de los servicios externos estaban en `.properties`.
- La autenticación mezclaba login de usuarios, autenticación entre servicios y firma electrónica.
  Las contrapartes salían como "externo" aunque el código las nombraba (WSDL, paquete, clave).

Principio rector: **análisis objetivo del código**. No se importa ningún inventario externo. Lo que
Traverse no puede ver en el código se declara como no visible, sin suponer nada.

## What Changes
- **Reconocimiento de estructura** (`traverse survey <ruta>` y paso previo automático de `scan`):
  - Recorre los niveles de directorios sin leer el código: solo nombres, tamaños y descriptores de build.
  - Localiza las raíces de código.
  - Detecta copias del mismo proyecto (mismas coordenadas de build, o mismo árbol de ficheros).
  - Elige la línea principal: `trunk`; si no existe, la de mayor versión; y explica por qué.
  - Clasifica lo que no es código (documentación, modelos, binarios). Los scripts SQL sí se incluyen.
  - Señala los módulos que solo tienen binarios.
  - `scan` analiza solo lo elegido y lo deja escrito en el informe. `--layout all` analiza todo y
    `--svn-all` queda como alias.
- **Integraciones declaradas en la configuración**:
  - Las URL de `.properties`, YAML, XML de Spring, `standalone.xml` y `server.xml` se convierten en
    contratos salientes según el esquema (http/https/ldap/jdbc/smtp/ftp/sftp).
  - Llevan transporte, host cifrado y la clave como pista de la contraparte.
- **Introspección de WAR, EAR y JAR** (solo lectura, sin ejecutar):
  - `web.xml` del artefacto: servlets, `security-constraint` y `CLIENT-CERT`.
  - Librerías de `WEB-INF/lib` y `lib/` para el SBOM y el cruce de vulnerabilidades.
  - Versión real del artefacto, comparada con la del descriptor.
- **Autenticación por tipo**:
  - Login de usuarios, autenticación entre servicios y firma electrónica, por separado.
  - Proveedores y filtros propios de Spring Security, preautenticación por cabeceras (SSO de pasarela),
    adaptadores Keycloak/SAML y `security-domain` del contenedor.
  - Reglas de acceso por ruta (`permitAll`/`authenticated`) cuando estén declaradas.
- **Clasificación de los endpoints entrantes**: API (REST/SOAP para otros sistemas) frente a
  pantallas de interfaz.
- **Contrapartes con nombre según el código** (servicio del WSDL, paquete del cliente, clave de
  configuración, nombre JNDI, esquema de BD, módulo de plataforma), con su evidencia:
  - Acceso a **esquemas de otros** como integración por datos.
  - **Módulos de plataforma** (módulos JBoss) como dependencias y condicionante de migración.
- **Menos ruido**:
  - Los alias de JPQL (`JOIN u.roles`) dejan de tomarse como `esquema.tabla`.
  - Los contenedores con clave privada (`.p12`, `.pfx`, `.jks`) pasan a severidad alta; un `.cer`
    queda como informativo.
  - La evidencia representativa prioriza el código frente a la configuración de logging.

## Impact
- Esquema de hechos 1.5.0, aditivo: `Repo.Layout` detallado, `Contract.Role` (api/ui),
  `Finding.AuthKind` y un bloque `survey` con la estructura.
- Cambia el comportamiento por defecto de `scan` en repos con copias: analiza menos y mejor, y lo
  declara en el informe.
- Sin dependencias nuevas. Sin red. Sigue sin ejecutar nada del repositorio analizado.
- Límites declarados:
  - Lo configurado en la infraestructura (balanceador, pasarela, servidor) no está en el código y se
    marca como "no visible en el código".
  - Los consumidores entrantes solo se conocen si también se analizan.
