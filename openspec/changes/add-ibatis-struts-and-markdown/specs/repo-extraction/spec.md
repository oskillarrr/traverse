## ADDED Requirements

### Requirement: Tablas desde mappers iBATIS/MyBatis
El extractor SHALL reconocer los mappers iBATIS (`<sqlMap>`) y MyBatis (`<mapper>`) por su elemento
raíz o DOCTYPE, con cualquier nombre de fichero, y SHALL extraer las tablas del SQL de cada
sentencia, registrando el namespace del mapper como componente.

#### Scenario: Mapper MyBatis con SQL dinámico
- **WHEN** un `<mapper namespace="es.x.dao.ArchivoDao">` tiene un `<select>` con `FROM ARCHIVO` y un `JOIN BENEFICIARIO` dentro de etiquetas dinámicas, y un `<insert>` con `INSERT INTO ARCHIVO`
- **THEN** se registran contratos `db-table` para ARCHIVO (lectura y escritura) y BENEFICIARIO (lectura), con componente `es.x.dao.ArchivoDao`

#### Scenario: Mapper iBATIS con nombre libre
- **WHEN** se analiza `resources/ibatis/archivo.xml` cuyo raíz es `<sqlMap namespace="ARCHIVO">`
- **THEN** el fichero se reconoce como descriptor y se extraen sus tablas

### Requirement: Acciones Struts por elemento raíz
El extractor SHALL reconocer la configuración Struts 2 (`<struts>`) y Struts 1 (`<struts-config>`)
por el elemento raíz, de modo que los ficheros de acciones incluidos con nombre arbitrario produzcan
endpoints HTTP entrantes.

#### Scenario: Fichero de acciones incluido
- **WHEN** se analiza `struts/ciudadano/altaSolicitud.xml` con raíz `<struts>` y un `<package namespace="/ciudadano">` con `<action name="altaSolicitud" class="es.x.Accion">`
- **THEN** se registra un contrato http/exposes `/ciudadano/altaSolicitud.action` con componente `es.x.Accion`

### Requirement: Aviso de múltiples raíces web
Cuando un módulo contiene más de un `WEB-INF/web.xml`, el sistema SHALL emitir un diagnóstico
`multiple-web-roots` que enumere las raíces.

#### Scenario: WAR con varios overlays
- **WHEN** un módulo tiene `ciudadano/WEB-INF/web.xml`, `gestor/WEB-INF/web.xml` y `webapp/WEB-INF/web.xml`
- **THEN** se emite un diagnóstico `multiple-web-roots` con las tres raíces
