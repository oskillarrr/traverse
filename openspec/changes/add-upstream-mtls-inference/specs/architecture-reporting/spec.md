## ADDED Requirements

### Requirement: Estado del mTLS entrante
Cada sistema que expone interfaces SHALL indicar el estado de su mTLS entrante:
- lo exige el servidor de aplicaciones;
- inferido en un intermediario, con el número de interfaces afectadas;
- no visible en el código.

`arc42.md` SHALL incluir un apartado ✍️ para los requisitos de transporte impuestos por terceros.

#### Scenario: Sin evidencia
- **WHEN** un sistema expone API y no hay ninguna señal de mTLS
- **THEN** el informe dice que el mTLS entrante no es visible en el código y que un intermediario
  podría exigirlo
