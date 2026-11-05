## ADDED Requirements

### Requirement: Base de evidencia de tablas compartidas
El agregador SHALL asignar confianza baja a las tablas compartidas que solo coinciden por nombre,
y media cuando coinciden el esquema o el datasource, y SHALL indicar la base (`schema`,
`datasource` o `name-only`).

#### Scenario: Mismo nombre en sistemas independientes
- **WHEN** dos sistemas sin esquema ni datasource común usan una tabla `USERS`
- **THEN** la relación `shared-db` tiene confianza `low` y base `name-only`

### Requirement: Llamadas internas
Una llamada que coincide con un endpoint expuesto por el mismo sistema SHALL considerarse interna
y SHALL NOT enlazarse con otros sistemas.

#### Scenario: Microservicios en un mismo repositorio
- **WHEN** el gateway llama a `/owners/{id}` y el servicio de clientes del mismo repositorio expone `/owners/{ownerId}`
- **THEN** no se crea relación con otro sistema que exponga `/owners/find`

### Requirement: Datasets mainframe compartidos
El agregador SHALL enlazar sistemas que usan el mismo dataset (DSN) con una relación `shared-dataset`.

#### Scenario: Fichero VSAM compartido
- **WHEN** un job de un sistema escribe `A.B.VSAM` y un programa CICS de otro lo lee
- **THEN** existe la relación `shared-dataset` entre ambos
