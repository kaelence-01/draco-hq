# 079 — Pegar en los campos de la TUI

**Fecha:** 2026-11-23
**Rama:** fix/pegar-en-campos → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

Al generar un documento, el autor no encontraba su commit en la lista de
Commits: había muchos de otras personas y el suyo había quedado fuera. Quiso
agregarlo con `h`, pero el campo no le dejaba pegar el hash, solo escribirlo.
Pidió dos cosas: poder pegar y un buscador de commits por hash, autor o fecha,
con un filtro para ver solo los suyos. Decidió hacerlo por partes: esta
actividad es el pegado; el buscador es la 080.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Lo pegado en el hash a mano se perdía | Había que copiar a mano un hash de 7 u 8 caracteres hexadecimales, con el riesgo de equivocarse, justo cuando el commit no está en la lista |
| Tampoco se podía pegar en los filtros `/`, la barra de comandos ni la ruta de Exportar | Pegar una clave, un comando copiado de la guía (`g -i OPS-7 -b a1b2c3d`) o una ruta no hacía nada, sin ningún aviso |

## 3. Diseño técnico

- **Causa.** Bubble Tea v2 activa el *bracketed paste*: lo pegado llega como
  un único `tea.PasteMsg`, no como una tecla por carácter. Los campos propios
  de draco (el hash a mano, `filter` y la barra) solo leían
  `tea.KeyPressMsg`. La ruta de Exportar es un `textinput` de Bubbles, que sí
  sabe pegar, pero la pantalla solo le pasaba las teclas. Los formularios de
  Huh ya funcionaban, porque reciben todos los mensajes.
- **Interfaz opcional `paster`** (`paste(text string)`) en el marco, como
  `captures` o `close`: `app` le pasa el `tea.PasteMsg` a la pantalla de
  arriba si la implementa. Si no, el mensaje llega a su `update`, que es como
  lo reciben un formulario de Huh o un `textinput`. Se eligió así y no un
  `case tea.PasteMsg` en el `update` de cada pantalla porque varias reparten
  las teclas desde `update` a funciones que solo reciben `tea.KeyPressMsg`
  (`press`, `handle`), y el pegado quedaría mezclado con ellas.
- **Fuera de un campo, pegar no hace nada.** Con la lista a la vista, pegar
  no abre el filtro ni la barra: si la terminal o el usuario pegan sin
  querer, nada cambia.
- **En una línea (`oneLine`).** Todos los campos son de una línea: las líneas
  de lo pegado se recortan y se unen con un espacio, y las vacías se
  descartan. Los espacios dentro de una línea se conservan, para no romper
  una ruta con dos espacios seguidos.
- **El hash a mano se queda con la primera palabra.** Así, pegar una línea de
  `git log --oneline` (`f1e2d3c Arregla el total de pagos`) deja solo el
  hash. El campo ya no aceptaba espacios al escribir.
- **La ruta de Exportar** le pasa el `tea.PasteMsg` al `textinput`, ya en una
  línea, solo mientras se puede editar (sin confirmación de reemplazo y antes
  de guardar).
- **Descartado:** cambiar los campos propios por `textinput`. Daría el
  pegado y el cursor dentro de la línea, pero `tui.md` §9 eligió los campos
  propios para que los tests no dependan del cursor parpadeante, y esta
  corrección no lo justifica.

## 4. Archivos

- `internal/tui/app.go` — interfaz opcional `paster`; el marco le entrega `tea.PasteMsg`.
- `internal/tui/filter.go` — `filter.paste` (solo mientras se escribe) y `oneLine`.
- `internal/tui/commits.go` — `paste`: la primera palabra, mientras se escribe el hash a mano.
- `internal/tui/cmdbar.go`, `internal/tui/home.go` — la barra de comandos recibe lo pegado cuando está abierta.
- `internal/tui/locations.go`, `profiles.go`, `projects.go`, `findings.go`, `export.go` — `paste` al filtro de cada lista; en `exportText`, `tea.PasteMsg` al `textinput` de la ruta.
- `internal/tui/paste_test.go` — nuevo: `oneLine`, hash pegado de una línea de `git log --oneline` hasta el documento, filtro, barra, tarea en el formulario de Huh y ruta de Exportar; fuera de un campo, pegar no hace nada.
- `docs/arquitectura/tui.md` — `paste` en la tabla de partes opcionales (§5), en las decisiones (§9) y en la norma de teclas (§10).
- `docs/guia-usuario.md` — dónde se puede pegar; el hash a mano acepta una línea de `git log --oneline`.
- `docs/roadmap.md` — sección «encontrar el commit» con 079 ✅ y 080 ⬜; `qa-listas` pasa a 081.
- `docs/bitacora/fase-7/079-pegar-en-campos.md`, índice, nota en la 078.

## 5. Cómo verificar

```bash
gofmt -l ./internal ./cmd                         # corrido: limpio
go vet ./internal/tui                             # corrido: limpio
go test ./internal/tui                            # corrido: ok
go test ./internal/tui -run 'Paste|OneLine' -v    # corrido: 5 tests ok
make check                                        # lo corre el autor
```

En una terminal real (lo corre el autor): `go run .`, Generar documento,
llegar a Commits, `h` y pegar un hash o una línea de `git log --oneline`.
Probar también pegar en un filtro `/` y en la barra de la principal.

## 6. Pendientes

- **Probar en terminales reales.** El pegado depende de que la terminal
  envíe *bracketed paste*: Windows Terminal, la de VS Code y las de Linux lo
  hacen. Los tests simulan el mensaje, no la terminal.
- **080:** el buscador de Commits, «solo los míos», cargar más, la búsqueda en
  todo el repo y los commits sugeridos por la clave de la tarea.

> **Nota (2026-11-23):** la 080 retiró la tecla `h`; un hash se pega en la
> búsqueda `/` y se agrega con `enter`. `paste` devuelve ahora un `tea.Cmd`.
