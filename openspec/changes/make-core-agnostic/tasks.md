# Tasks

- [x] 1. Campo `authKind` en las reglas, validado al cargar; el hallazgo lo hereda de su regla
- [x] 2. Quitar la tabla de tipos del código (solo quedan los `DESC-*` del motor)
- [x] 3. Catálogo 2.0.0: fuera `GOV-*`; nuevas `SIGN-*` y `AUTH-EIDAS` con ejemplos
- [x] 4. Paquete `rulepacks/es-public-sector.json` (`ES-*`) generado por `tools/genrules.py`
- [x] 5. Clave `rules` en `traverse.json`, con rutas relativas al fichero
- [x] 6. Test de paquetes: cargan, pasan sus ejemplos y no sustituyen reglas del núcleo
- [x] 7. Certificados de persona física por X.520 y ETSI EN 319 412-1, con tests de varios países
- [x] 8. Vocabulario de entornos y carpetas contenedoras en varios idiomas
- [x] 9. Nombres neutros en ejemplos, tests, especificaciones y documentación
- [x] 10. Demo regenerada, README, manual, docs/rules.md y CHANGELOG
- [x] 11. Revalidar en un entorno real con el paquete `es-public-sector` cargado (0.9.0 y 0.9.1)
