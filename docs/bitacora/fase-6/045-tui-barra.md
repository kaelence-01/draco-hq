# 045 — Barra de comandos (1): descripción de los comandos, sugerencias y Ayuda

**Fecha:** 2026-10-21
**Rama:** feature/tui-barra → develop
**Fase del roadmap:** 6

## 1. Solicitud

Seguir con la actividad 5a del plan de la fase 6 (bitácora 041): la
descripción de los comandos y su parser fuera de `cmd`, la barra con
autocompletado y la pantalla Ayuda.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Nombres, flags y parser de los comandos vivían en `cmd`, que la TUI no importa | La barra habría necesitado su propia tabla de flags, y los dos modos leerían una línea distinto |
| Los flags no tenían descripción | Una sugerencia sin descripción obliga a salir a `draco help` para saber qué hace cada flag |
| La ayuda de cada comando solo existía como texto del modo comando | Una Ayuda en la TUI habría copiado esos textos, y se desactualizarían por separado |

## 3. Diseño técnico

- **`internal/cli`** (nuevo):
  - `Flag` con `Desc`;
  - `Command` con nombres, resumen, flags y subcomandos;
  - `Find` por nombre o alias;
  - `Parse`, que es el parser de `cmd` sin cambios de comportamiento (sus
    tests se movieron con él).
- **`cmd`:**
  - usa `cli.Flag` y `cli.Parse` directamente. Los cambios mecánicos
    (`parseFlags` → `cli.Parse`, `f.get` → `f.Get`, `f.args` → `f.Args`)
    se hicieron con `sed`, y `flags.go` se eliminó;
  - cada comando se registra con `flags` y `subs`. Los flags inline de
    `template init` y `update` pasan a variables (`templateInitFlags`,
    `updateFlags`), y `project` y `profile` comparten sus subcomandos
    (`configSubs`);
  - `commandSpecs()` y `helpText(name)` llegan a la TUI por `Deps.Commands`
    y `Deps.Help`;
  - un test nuevo exige que cada flag y subcomando tenga descripción y
    aparezca en la ayuda de su comando (con la forma `-i, --issue`, si
    tiene corto). Los tests de los comandos pasaron sin cambios.
- **Barra** (`cmdbar.go`):
  - `barWords`, `suggest`, `complete` y `readLine` son funciones puras,
    probadas con tablas;
  - `help` es propio de la barra, porque el modo comando lo atiende antes
    de su router;
  - se abre con `/` en la principal y se queda con todas las teclas;
  - mientras está abierta, la principal oculta el menú y el panel. Sin eso,
    en 80×24 el marco recortaba la barra por abajo (lo detectó el test de
    80×24);
  - el campo es texto propio, como el filtro de las listas: no tiene el
    cursor parpadeante de `textinput`, ni mueve el cursor dentro de la
    línea.
- **`enter` en esta actividad:**
  - un error de lectura (flag, comando o subcomando desconocido, valor que
    falta) se muestra en la barra;
  - `help [comando]` abre la Ayuda y `version` responde en la línea;
  - los demás comandos dicen que la línea es válida y que se elijan en el
    menú. Abrirlos es la actividad 5b.
- **Ayuda** (`helpscreen.go`):
  - la general explica qué es draco, las teclas globales, la barra y los
    comandos;
  - la de un comando es el texto de `draco help`, en un `viewport`, con el
    nombre canónico en el título.

## 4. Archivos

- `internal/cli/cli.go`, `cli_test.go` — nuevos (los tests del parser,
  movidos desde `cmd/flags_test.go`, y `Find`).
- `cmd/flags.go`, `cmd/flags_test.go` — eliminados.
- `cmd/app.go` — `command` con `flags`, `subs` y `names`.
- `cmd/help.go` — `commandSpecs` y `helpText`.
- `cmd/generate.go`, `qa.go`, `fetch.go`, `backlog.go`, `template.go`,
  `update.go`, `project.go`, `profile.go` — `cli.Flag` con `Desc`,
  `cli.Parse`, `flags` y `subs` al registrar.
- `cmd/visual.go` — `Commands` y `Help` en `tui.Deps`.
- `cmd/app_test.go` — cada flag y subcomando, descrito y en su ayuda.
- `internal/tui/cmdbar.go`, `helpscreen.go`, `cmdbar_test.go` — nuevos.
- `internal/tui/home.go` — la barra en la principal.
- `internal/tui/tui.go` — `Deps.Commands` y `Deps.Help`.
- `CLAUDE.md` — `internal/cli` en la arquitectura y las reglas del modo
  comando con `cli.Flag`.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  diseño ajustado a lo implementado.
- `docs/bitacora/fase-6/045-tui-barra.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/cli
make test pkg=./cmd run=EveryFlag v=1
make test pkg=./internal/tui run='Bar|Suggest|Complete|ReadLine' v=1
make check
make run            # / en la principal: sugerencias, tab, /help g, /g --x
```

## 6. Pendientes

- Actividad 5b: que `enter` abra la pantalla de cada comando con los datos
  cargados.
- Mejoras posibles: mover el cursor dentro de la línea y escribir valores
  con espacios entre comillas.
