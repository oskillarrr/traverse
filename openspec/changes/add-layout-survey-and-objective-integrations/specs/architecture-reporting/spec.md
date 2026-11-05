## ADDED Requirements

### Requirement: Estructura del repositorio en el informe
El informe de cada sistema SHALL mostrar qué se analizó y qué se omitió:
- raíces de código;
- línea principal y motivo de la elección;
- copias omitidas, con su versión y si son idénticas;
- contenido que no es código.

#### Scenario: Informe de un repositorio con entregas
- **WHEN** se analiza una ruta con `trunk/` y 26 entregas
- **THEN** la sección "Estructura del repositorio" indica "Analizado: trunk (2.10.2)", lista las copias omitidas y el resumen ejecutivo lo menciona

### Requirement: Autenticación e interfaces separadas
La sección de seguridad SHALL separar login de usuarios, autenticación entre servicios y firma
electrónica, e indicar "no visible en el código" cuando no haya evidencia. El catálogo de interfaces
expuestas SHALL distinguir la API (para otros sistemas) de las pantallas de interfaz.

#### Scenario: Sin evidencia de autenticación de usuarios
- **WHEN** no hay ningún mecanismo de login en el código
- **THEN** el informe dice "Login de usuarios: no visible en el código (posiblemente delegado en la infraestructura)"

#### Scenario: API frente a pantallas
- **WHEN** un sistema expone 14 rutas en un `@RestController` y 193 en controladores con vista
- **THEN** el resumen indica "API: 14 · Interfaz: 193"
