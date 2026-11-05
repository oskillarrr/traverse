# Design

## 1. Reconocimiento de estructura (survey)
Es una pasada barata antes del análisis: recorre el árbol leyendo solo nombres, tamaños y los
descriptores de build. Por cada directorio, hasta una profundidad configurable (por defecto 4), calcula:

- Ficheros y bytes por categoría: código, build, configuración, SQL, documentación, binario desplegable, otros.
- Descriptores de build presentes, con sus coordenadas y versión: `pom.xml` (groupId:artifactId:version),
  `build.gradle`, `*.csproj`, `package.json`, `web.xml`.
- Binarios desplegables (`.war`, `.ear`, `.jar` fuera de `lib/`) y si hay fuentes junto a ellos.

### Detección de copias
Dos raíces son **copias del mismo proyecto** cuando:
1. comparten coordenadas del descriptor raíz (por ejemplo, `org.example:orders` en distintas carpetas); o
2. si no hay descriptor, la similitud de Jaccard de sus rutas relativas es mayor o igual que 0,8.

Si además coinciden las rutas y los tamaños, se marcan como **idénticas** (por ejemplo,
"entregas/2.10.2 = trunk").

### Elección de la línea principal
Orden de preferencia:
1. Una copia bajo `trunk`.
2. La de mayor versión (semver del descriptor).
3. La de ruta más corta.

Siempre se registra la decisión con su motivo: "trunk (2.10.2); develop 2.11.0 y 26 entregas omitidas".

### Lo que no es código
Se clasifica por contenido, no por nombre:
- Mayoría de `.doc/.docx/.pdf/.xls` → documentación.
- `.eap/.qea/.vpp/.archimate` → modelos.
- Solo binarios → binarios.

Los `.sql` siempre se incluyen, porque aportan el modelo de datos.

### Salida
- `traverse survey <ruta>` imprime el árbol anotado por niveles y escribe `survey.json` y `survey.md`.
- `scan` usa el mismo motor y añade la sección "Estructura del repositorio" al informe.

## 2. URL declaradas en la configuración
- **Parser clave=valor** para `.properties`. **Recorrido de valores** para YAML.
- **En XML** se leen:
  - atributos y texto con URL en `<property value>`, `<entry>` y `<value>`;
  - `<connection-url>` (datasource);
  - `<smtp-server>` y `outbound-socket-binding`.
- **Por esquema**:
  - `http/https` → contrato `http calls`;
  - `jdbc:` → `database` (host y servicio cifrados, con el esquema si viene en la URL);
  - `ldap/ldaps` → `ldap`;
  - `smtp/smtps` → `mail`;
  - `ftp/sftp` → `file-transfer`.
- **Pista de contraparte**: el prefijo de la clave (`signer.url` → "signer").
- Confianza media, porque una URL configurada puede no usarse. Si se usa desde el código, sube.

## 3. Introspección de artefactos
Se abre el zip con `archive/zip` de la librería estándar, sin descomprimir a disco y con límites de
tamaño y número de entradas (protección frente a zip bombs).

- Se leen `WEB-INF/web.xml`, `META-INF/application.xml`, `jboss-web.xml` y `jboss-deployment-structure.xml`.
- Se leen `META-INF/maven/**/pom.properties` de cada JAR anidado (un solo nivel) para el SBOM.
- Nunca se cargan clases.
- Los hallazgos se atribuyen al artefacto con la ruta `x.war!/WEB-INF/web.xml`.

## 4. Autenticación por tipo
`Finding.AuthKind` toma uno de estos valores: `user`, `service`, `signature`, `transport`.

Señales:
- Clases que extienden `AbstractAuthenticationProcessingFilter`, implementan `AuthenticationProvider`
  o son subclases de `WebSecurityConfigurerAdapter` → `user` / custom, con el nombre de la clase.
- `RequestHeaderAuthenticationFilter` o `J2eePreAuthenticatedProcessingFilter` → SSO delegado a
  pasarela o contenedor.
- `keycloak.json`, `keycloak-saml.xml` o `auth-method` KEYCLOAK/KEYCLOAK-SAML → Keycloak.
- `security-domain` en `jboss-web.xml` → contenedor.
- AutoFirma en pantallas o controladores de login → `user` (certificado). En el resto de casos → `signature`.
- Reglas de acceso: `antMatchers/requestMatchers(...).permitAll()/authenticated()/hasRole()` → tabla
  de rutas protegidas y públicas.

## 5. API frente a interfaz
- **API**: `@RestController`, `@ResponseBody`, JAX-RS, servicios SOAP, rutas bajo `/ws`, `/api`,
  `/rest`, `/services`.
- **Interfaz**: controladores MVC que devuelven vista, JSP y Struts actions con `forward` a JSP.
- El rol se añade al contrato.

## 6. Contrapartes desde el código
El nombre se toma, por este orden, de:
1. el servicio del WSDL o su `targetNamespace`;
2. el segmento significativo del paquete del cliente (`orders.negocio.payrollWS` → "payroll");
3. el prefijo de la clave de configuración;
4. el nombre JNDI;
5. el esquema de BD ajeno.

Se acompaña siempre de su evidencia. No hay mapeo externo.
