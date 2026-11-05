# Design: núcleo agnóstico

## Principio
El núcleo reconoce **estándares, frameworks abiertos y productos de cualquier mercado**. El
conocimiento de un país, sector u organización es **dato opcional**: un fichero de reglas que el
usuario carga a propósito. Así el binario es el mismo para todos y lo regional se versiona y se
prueba por separado.

## Decisiones
- **Paquetes como ficheros, no embebidos.** Un paquete embebido seguiría siendo parte del producto.
  Al quedar en `rulepacks/`, se distribuye con las fuentes, se audita como cualquier fichero de
  reglas y no cambia el binario.
- **Prefijo de ámbito obligatorio** (`ES-`, `ACME-`…). Un paquete nunca sustituye una regla del
  núcleo por accidente; el test `TestRulePacks` lo comprueba. Sustituir a propósito sigue siendo
  posible con un fichero de reglas propio y el mismo id.
- **`authKind` en la regla.** La tabla en el código obligaba a tocar Go para cada regla nueva y
  acoplaba el motor a ids concretos. Ahora una regla propia aparece sola en el apartado correcto
  (login, entre servicios, firma, transporte). Los hallazgos `DESC-*` los crea el motor a partir de
  descriptores Java EE, que son estándar, y conservan su tipo en el código.
- **Persona física sin convenciones nacionales.** Se usan los atributos de X.520 y los
  identificadores semánticos de ETSI EN 319 412-1 (`PAS`, `IDC`, `PNO`, `TAX`, `TIN` + código ISO
  3166). Como red de seguridad, también se trata como personal un CN con un número de 6 o más
  dígitos o un correo en un sujeto que no es un host. Ocultar un sujeto de más es el error seguro.
- **Vocabulario multilingüe, no nacional.** Las palabras de capas, entornos y carpetas en varios
  idiomas no son conocimiento de un país. Se mantienen, se amplían y siguen siendo configurables
  (`layers.tokens`).
- **`traverse.json` admite `rules`**, con rutas relativas al fichero. Así un portfolio fija sus
  paquetes una vez. Orden de carga: núcleo → `rules` de la configuración → `--rules`.

## Alternativas descartadas
- **Borrar las reglas españolas sin más.** Se perdería valor real para quien las usa, sin ganar
  nada en el núcleo.
- **Flag `--pack <nombre>` con paquetes embebidos.** Más cómodo, pero devuelve lo regional al
  binario.
