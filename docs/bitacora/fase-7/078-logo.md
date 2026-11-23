# 078 — Logo nuevo en la pantalla principal

**Fecha:** 2026-11-17
**Rama:** feature/logo → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

El autor quería un logo para draco y que la pantalla principal de la TUI
dejara de mostrar «DRACO» en letras de bloque y mostrara el dragón de su
lámina de marca (un recorte ASCII/ANSI y el símbolo aislado, ambos fuera del
repo). Pidió probarlo directamente en el CLI. Al verlo, pidió que el logo
se mantuviera en cada vista, con sus mismas dimensiones y sin afectar a las
demás pantallas.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La pantalla principal mostraba solo el nombre, sin la identidad de la marca | La herramienta se ve igual que cualquier otra de terminal |
| El logo de marca es una imagen y la mayoría de las terminales no pinta imágenes | Hay que traducirlo a caracteres (ver §3) |

## 3. Diseño técnico

- **Fuente:** el símbolo aislado de la lámina de marca (cabeza, cuerno y
  cuello del dragón). Una prueba con el recorte ASCII dio peor resultado:
  ya venía rebajado a cuadros y las diagonales finas se perdían.
- **Conversión con un script de Pillow, fuera del repo.** Se borran la
  constelación y el punto cian, se recorta la cabeza y se pasa a braille
  (cada carácter son 2×4 puntos) con un umbral de luminosidad. Los espacios
  en blanco del braille se guardan como espacios normales, para no dejar
  caracteres invisibles en el código.
- **Descartado:**
  - **Pintar el PNG** con protocolos de imagen (Kitty, iTerm2, Sixel): no es
    fiable en WSL, SSH ni tmux, y rompe la norma de estilos ANSI básicos.
  - **Semibloques** (`▀ ▄ █`): se probaron en seis tamaños y a 6 u 8 líneas
    dan una mancha.
  - **Cabeza con cuello en S** (10 líneas): conserva mejor el símbolo pero es
    demasiado alta para la principal; queda para otra pantalla si hace falta.
  - **Dibujarlo a mano:** el primer intento quedó lejos de la imagen.
- **Resultado:** cabeza de 6 líneas por 13 columnas, una barra `│` y el
  nombre `d r a c o`, 29 columnas en total. Va con `logoStyle`, como el logo
  anterior.
- **El umbral de ancho** ahora se calcula con `lipgloss.Width` de todo el
  logo (`logoWidth`). Antes usaba la longitud de la primera línea, que en el
  logo nuevo es la más corta. Con 29 columnas, el logo se muestra desde 33
  columnas; antes, desde 45.
- **Altura:** una línea más que el logo anterior. La principal ya retira el
  eslogan y las descripciones cuando falta altura, así que no cambió su
  lógica.
- **No incluidos:** la constelación y el punto cian del símbolo.
- **El logo vive en el marco**, no en cada pantalla: `app.header` lo dibuja y
  las pantallas solo reciben el cuerpo que queda, como siempre. Ninguna
  cambia.
  - **Elástico.** El logo ocupa 6 líneas y la raya una más; las pantallas
    están diseñadas para 20 líneas de cuerpo (80×24). El marco lo muestra solo
    si, con el pie en sus 2 líneas, el cuerpo conserva esas 20: desde 29 líneas
    (`minBody`, `footerLines`) y 63 columnas (`bannerInfoWidth`). Con menos,
    queda la cabecera de una línea de antes.
  - **Con la ayuda completa abierta** el pie crece y el cuerpo se reduce, como
    ya pasaba; el logo no desaparece por eso, porque el umbral se mide con el
    pie cerrado.
  - **Al lado del logo** van la versión, la ruta, el perfil y el proyecto, en
    líneas aparte. El perfil y el proyecto ya no comparten línea porque no
    caben a 63 columnas.
  - **La principal** no dibuja entonces el suyo: el marco se lo dice con
    `state.banner` y ella conserva su logo propio cuando el marco no lo muestra.
- **Descartado:** el logo siempre visible (a 80×24 el cuerpo se quedaría en
  15 líneas y habría que rediseñar varias pantallas) y un mini-logo de dos
  líneas con el completo solo en la principal.
- **Decisiones del autor:**
  - que el logo salga en la principal y que lo vea en una terminal real;
  - el comportamiento elástico del marco, elegido entre las tres opciones
    anteriores.

## 4. Archivos

- `internal/tui/logo.go` — `logoLines` con la cabeza en braille, la barra y el nombre; `logoWidth`; `logo(width)` usa el ancho de todo el bloque.
- `internal/tui/app.go` — `hasRoomForBanner`, `bannerHeader`; `header` se parte en `version`, `titles`, `profileText` y `projectText`, que usan las dos cabeceras; `render` fija `state.banner`.
- `internal/tui/tui.go` — `state.banner`.
- `internal/tui/home.go` — sin logo propio cuando el marco muestra el suyo (`view`, `wide`, `narrow`).
- `internal/tui/logo_test.go` — nuevo: cabe en 80 columnas, la barra está en la misma columna y todo mide una columna, el respaldo `draco` sin ancho suficiente, el logo sale una vez en cualquier pantalla con espacio, y el cuerpo conserva sus 20 líneas.
- `internal/tui/app_test.go` — la principal muestra el dragón y el nombre; la terminal angosta (`logoWidth+3`) muestra solo `draco`; el marco a pantalla completa vale con y sin logo.
- `internal/tui/history_test.go` — la prueba de Recientes pasa de 160×40 a 160×41: la línea extra del logo dejaba tres filas en vez de cuatro.
- `internal/tui/qa_test.go` — la terminal de los tests de QA pasa de 110×34 a 110×39, para conservar el mismo cuerpo que antes de que el marco llevara el logo.
- `docs/arquitectura/tui.md` — mapa de archivos, el logo en el marco, la principal y su degradación.
- `docs/guia-usuario.md` — el logo en la cabecera de todas las pantallas, desde 29 líneas y 63 columnas.
- `docs/roadmap.md` — 078 ✅; `qa-listas` pasa a 079.
- `docs/bitacora/fase-7/078-logo.md`, índice, nota en la 077.

## 5. Cómo verificar

```bash
gofmt -l internal/tui                  # corrido: limpio
go vet ./internal/tui                  # corrido: limpio
go test ./internal/tui                 # corrido: ok
go vet ./... && make check             # lo corre el autor
```

Con una terminal real (lo corre el autor): `go run .`.

- **Con 29 líneas o más y 63 columnas o más:** el logo en la cabecera de la
  principal y de cualquier pantalla (Proyectos, Configuración, un formulario).
- **Con menos alto o ancho:** la cabecera de una línea de antes; la principal
  conserva su logo propio mientras el ancho lo permita.
- **Ensanchar o estrechar la ventana** con la TUI abierta: el logo aparece y
  desaparece sin romper ninguna pantalla.

## 6. Pendientes

- **Ver el logo en una terminal real**, en la principal y en otras pantallas.
  Los tests miden anchos y posiciones, pero no la fuente: el braille depende
  de ella.
- **Pantallas con mucho contenido a 29 líneas.** El cuerpo queda justo en 20,
  lo que las pantallas ya soportan a 80×24, pero no se probó una por una con
  el logo.
- **No hay respaldo por fuente.** Si la fuente de una terminal no tiene
  braille, el logo sale con huecos; el respaldo es solo por ancho.
- Los archivos de marca (SVG o PNG) no están en el repo, ni el logo en el
  README o en `draco version`.
- **079 (opcional):** módulo y sprint de QA elegidos de listas.

> **Nota (2026-11-23):** la 079 es ahora `fix/pegar-en-campos` y la 080 el
> buscador de Commits; las listas de QA pasan a la 081.
