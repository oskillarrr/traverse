## ADDED Requirements

### Requirement: Reconocimiento de estructura antes del análisis
El sistema SHALL recorrer los niveles de directorios de cada ruta analizada para:
- localizar las raíces de código;
- detectar copias del mismo proyecto (mismas coordenadas de build o árbol de ficheros similar);
- elegir una línea principal (trunk, si no la de mayor versión, si no la de ruta más corta);
- clasificar el contenido que no es código.

Por defecto, el análisis SHALL limitarse a la línea principal y SHALL dejar registradas las copias
omitidas y el motivo de la elección.

#### Scenario: Copias de versión con nombre no estándar
- **WHEN** una ruta contiene `trunk/`, `develop/` y `entregas/<versión>/` con el mismo `groupId:artifactId`
- **THEN** solo se analiza `trunk`, y el informe lista `develop` y cada entrega como copias omitidas, con su versión

#### Scenario: Copia idéntica a la línea principal
- **WHEN** una copia tiene exactamente las mismas rutas relativas y tamaños que la línea principal
- **THEN** se marca como idéntica (por ejemplo, "entregas/2.10.2 = trunk")

#### Scenario: Código en una subcarpeta junto a documentación
- **WHEN** la raíz contiene `doc/`, `ea/` y `bbdd/` sin código y `fuente/` con proyectos Maven
- **THEN** se analiza el código de `fuente/` y los `.sql` de `bbdd/`, y `doc/` y `ea/` se clasifican como documentación y modelos

#### Scenario: Variantes por entorno
- **WHEN** `config/int`, `config/pre` y `config/pro` contienen los mismos nombres de fichero
- **THEN** no se tratan como copias y se analizan las tres

#### Scenario: Runtime incluido en el repositorio
- **WHEN** el repositorio contiene `Liberty_Settings/opt/jdk/jdk21.0.5_11/lib/...`
- **THEN** esa carpeta se omite del análisis y se informa como runtime incluido

#### Scenario: Ramas SVN con estructura antigua
- **WHEN** `fuente/branches/` contiene ramas con módulos distintos de los de `trunk`
- **THEN** todas las ramas se omiten como copias de `fuente/trunk`

#### Scenario: Analizar todo de forma explícita
- **WHEN** se usa `--layout all` (o `--svn-all`)
- **THEN** se analizan todas las copias como hasta ahora

### Requirement: Integraciones declaradas en configuración
El sistema SHALL convertir en contratos salientes las URL declaradas en ficheros de configuración
(`.properties`, YAML, XML de Spring y descriptores de servidor). Cada contrato SHALL llevar:
- el tipo según el esquema;
- el transporte;
- el host cifrado;
- la clave como pista de la contraparte.

#### Scenario: Endpoint en properties
- **WHEN** `config.properties` contiene `signer.url=https://host/signerws/services/DSSSignerVerify`
- **THEN** se registra un contrato saliente `http` con transporte `tls`, contraparte sugerida "signer" y evidencia en fichero y línea

#### Scenario: Datasource JDBC
- **WHEN** un descriptor declara `jdbc:oracle:thin:@host:1521/SERVICIO`
- **THEN** se registra un contrato de base de datos con el host cifrado y el servicio

### Requirement: Introspección de artefactos desplegables
El sistema SHALL leer, sin ejecutarlos ni descomprimirlos a disco, los WAR, EAR y JAR desplegables
presentes en el repositorio y extraer:
- sus descriptores;
- las restricciones de seguridad (incluido `CLIENT-CERT`);
- las librerías embebidas (para SBOM y vulnerabilidades);
- la versión real del artefacto.

#### Scenario: Módulo entregado solo como WAR
- **WHEN** un módulo contiene `sign-gateway-2.4-INT.war` y un `pom.xml` de versión 2.3, sin fuentes
- **THEN** se analizan el `web.xml` y las librerías del WAR, y se registra un diagnóstico de divergencia de versión (2.3 frente a 2.4-INT)

### Requirement: Autenticación clasificada por tipo
Cada hallazgo de autenticación SHALL indicar si corresponde a login de usuarios, autenticación entre
servicios, firma electrónica o seguridad de transporte. Los proveedores y filtros de autenticación
propios SHALL identificarse por su nombre de clase.

#### Scenario: Proveedor propio de Spring Security
- **WHEN** una clase implementa `AuthenticationProvider`
- **THEN** se registra un mecanismo de login de usuarios "propio" con el nombre de la clase

#### Scenario: AutoFirma en la pantalla de login
- **WHEN** `login.jsp` usa el cliente de @firma
- **THEN** se clasifica como login de usuarios con certificado, no como firma de documentos

### Requirement: Contrapartes nombradas según el código
Toda integración saliente SHALL llevar una contraparte deducida solo de evidencias del propio código
(servicio del WSDL, paquete del cliente, clave de configuración, nombre JNDI, esquema de BD ajeno), sin
datos importados. Los accesos a tablas de esquemas distintos del propio SHALL registrarse como
integración por datos.

#### Scenario: Cliente SOAP generado
- **WHEN** `orders.negocio.payrollWS.ServiciosTurnosImplService` es un `@WebServiceClient`
- **THEN** la contraparte es "payroll" (por el paquete) y el servicio es `ServiciosTurnos`

### Requirement: Menos falsos positivos en el acceso a datos
Los alias y las rutas de navegación de JPQL/HQL (`JOIN u.roles`) SHALL NOT registrarse como tablas
`esquema.tabla`.

#### Scenario: Join JPQL
- **WHEN** una consulta es `select r from Usuario u join u.roles r`
- **THEN** no se registra la tabla `U.ROLES`
