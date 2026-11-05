# portfolio-aggregation Specification

## Purpose
Link the facts of several repositories into a portfolio: libraries, interface contracts, shared data and metrics.

## Requirements

### Requirement: Ejecución por lotes desde manifiesto
El sistema SHALL aceptar un manifiesto `repos.csv` (`name, path, group`) y
SHALL ejecutar la extracción de cada repositorio de forma independiente, de modo que el fallo
de uno no detenga los demás.

#### Scenario: Un repositorio ilegible
- **WHEN** la ruta de un repositorio del manifiesto no existe
- **THEN** se informa el fallo en la consola y el resto del lote continúa

### Requirement: Grafo de dependencias entre repositorios
El agregador SHALL enlazar repositorios cuando un módulo de uno declara dependencia sobre un GAV
producido por un módulo de otro, ignorando la versión para el enlace y reportando divergencias
de versión entre consumidores.

#### Scenario: Librería común
- **WHEN** `app-a` y `app-b` dependen de `org.example:comun-dto` producido por `lib-comun`
- **THEN** el landscape muestra `app-a → lib-comun` y `app-b → lib-comun`
- **AND** si consumen versiones distintas se lista la divergencia

### Requirement: Matching de contratos productor-consumidor
El agregador SHALL emparejar contratos `consumed` con `provided` del mismo `kind` por clave
normalizada, SHALL asignar `confidence` según el tipo de coincidencia y SHALL listar los
contratos consumidos sin productor conocido como **dependencias externas al portfolio**.

#### Scenario: Cola JMS compartida
- **WHEN** `app-a` produce en `jms/queue/Expedientes` y `app-b` declara un MDB sobre el mismo destino
- **THEN** existe la relación `app-a → app-b` de tipo `jms` con `confidence=alta`

#### Scenario: Ruta demasiado genérica
- **WHEN** la ruta expuesta tiene menos de dos segmentos literales (p. ej. `/{id}`)
- **THEN** no se crea relación automática

#### Scenario: Coincidencia más específica
- **WHEN** una llamada a `/notifications/api/messages/send` coincide con `POST /api/messages/send` y con `GET /api/messages/{id}`
- **THEN** solo se enlaza con `POST /api/messages/send`

### Requirement: Matriz de propiedad de datos
El agregador SHALL producir una matriz aplicación × tabla con el tipo de acceso inferido y
SHALL señalar las tablas accedidas por más de una aplicación.

#### Scenario: Base de datos compartida
- **WHEN** `app-a` y `app-c` referencian la tabla `EXPEDIENTE`
- **THEN** `EXPEDIENTE` aparece marcada como **compartida** en el informe de portfolio

### Requirement: Métricas y ciclos
El sistema SHALL calcular Ca, Ce e inestabilidad por paquete, y SHALL detectar ciclos
(componentes fuertemente conexos) entre paquetes y entre módulos de build.

#### Scenario: Ciclo entre módulos
- **WHEN** `modulo-a` depende de `modulo-b` y `modulo-b` de `modulo-a`
- **THEN** el ciclo se lista con las aristas y evidencias que lo forman
