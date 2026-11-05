# Design

## Reconocimiento por contenido
`IsDescriptor` detecta, además de `<beans`, los marcadores `<struts`, `<sqlMap`, `<sqlMapConfig`,
`<mapper` y los DOCTYPE de iBATIS/MyBatis en los primeros 2 KB. `Parse` despacha por el nombre del
elemento raíz, no por el nombre de fichero, de modo que los ficheros incluidos desde `struts.xml` o
los mappers con nombres libres se procesan igual.

## Tablas desde mappers
Para cada sentencia del mapper se toma su texto interno —incluido el de etiquetas dinámicas
(`<if>`, `<where>`, `<foreach>`, `<include>`, `<sql>`)— con `Node.InnerText`, y se pasa por
`sqlparse.Tables`. El acceso (lectura/escritura) lo decide el verbo SQL, no la etiqueta. El
namespace del mapper se registra como componente. Los duplicados (una tabla en varias sentencias, o
en fragmentos `<sql>` reutilizados) se colapsan luego en el deduplicado de contratos.

## sqlparse
La lógica de detección de tablas (expresiones FROM/JOIN/INTO/UPDATE, verbo de escritura, lista de
ruido) se mueve a `internal/sqlparse`, para que el escáner de código (literales SQL, scripts, EXEC
SQL) y el de descriptores compartan exactamente las mismas reglas.

## Múltiples raíces web
No se inventa un context root por variante (sería adivinar). Se emite un diagnóstico que lista las
raíces encontradas y recomienda verificar; el context root inferido del WAR se mantiene como hasta
ahora y los endpoints se marcan como "(context root inferred)".

## Resumen y Markdown
`summaryLines` y `portfolioSummary` derivan las cifras de los hechos y del portfolio. `report.md`
reutiliza el catálogo de integraciones (`integrations`) y el constructor Markdown de arc42, pero sin
marcadores "por completar": es el equivalente en texto de `report.html`.

## Alternativas descartadas
- Resolver `<include refid>` de MyBatis uniendo fragmentos: innecesario, porque los fragmentos `<sql>`
  se escanean por separado y el deduplicado colapsa lo repetido.
- Separar el WAR multi-overlay en varios desplegables con context roots adivinados: se descartó por
  no fabricar datos; se avisa con un diagnóstico.
