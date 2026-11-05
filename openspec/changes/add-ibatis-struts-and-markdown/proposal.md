# Change: extractores iBATIS/MyBatis y Struts, y exportación a Markdown (v0.5.0)

## Why
La verificación con un proyecto legacy real (una aplicación Java/JBoss de administración, WAR único
con iBATIS, Struts 2 y Spring) mostró tres huecos y dos carencias de informe:

1. El modelo de datos quedaba casi invisible: con iBATIS/MyBatis, el SQL vive en los mappers XML, y
   Traverse solo extraía tablas de literales SQL en Java (2 tablas de una aplicación con decenas).
2. La superficie de entrada HTTP estaba subestimada: las acciones Struts 2 se definen en ficheros
   incluidos con nombres arbitrarios (no `struts.xml`), que Traverse no reconocía.
3. Un mismo módulo con varias raíces web (overlays `ciudadano/`, `gestor/`, `webapp/`) se colapsaba
   en un WAR sin avisar de las variantes.
4. Faltaba un resumen ejecutivo que diera la foto en diez líneas.
5. Faltaba exportar el análisis completo a Markdown (el HTML no es cómodo de versionar ni compartir
   en texto).

## What Changes
- **iBATIS/MyBatis**: se reconocen `sqlMap`, `mapper` y `sqlMapConfig` por elemento raíz o DOCTYPE
  (con cualquier nombre de fichero) y se extraen las tablas del SQL de cada sentencia
  (`select`/`insert`/`update`/`delete`/`statement`/`procedure`/`sql`), incluido el SQL dentro de
  etiquetas dinámicas. El namespace del mapper (la interfaz DAO en MyBatis) queda como componente.
- **Struts 2 y 1 por contenido**: se reconocen por el elemento raíz `<struts>` / `<struts-config>`,
  de modo que los ficheros de acciones incluidos (nombre arbitrario) producen endpoints entrantes.
- **Múltiples raíces web**: un módulo con más de un `WEB-INF/web.xml` emite el diagnóstico
  `multiple-web-roots` para que el arquitecto verifique el context root de cada variante.
- **Resumen ejecutivo**: en el informe HTML (por sistema y portfolio), en `report.md` y en
  `portfolio.md`: tamaño y stack, despliegue, integraciones, vulnerabilidades, riesgos y acoplamiento.
- **Exportación a Markdown**: nuevo `report.md` por sistema, un informe analítico poblado (sin
  secciones "por completar") que refleja el contenido de `report.html`.
- Refactor: la extracción de tablas desde SQL pasa a `internal/sqlparse`, compartida por el escáner
  de código y el de descriptores.

## Impact
- Más contratos `db-table` y `http/exposes`; la cifra de tablas y de endpoints sube de forma acorde
  a lo que hay en el código.
- Nuevo fichero `report.md` por sistema. Sin cambios en los formatos existentes.
- Esquema de hechos sin cambios (1.3.0). Sin dependencias nuevas.
