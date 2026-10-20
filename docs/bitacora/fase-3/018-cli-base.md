# 018 — Base del modo comando

**Fecha:** 2026-10-19
**Rama:** feature/cli-base → develop
**Fase del roadmap:** 3

## 1. Solicitud

Seguir con la fase 3: la base del comando (actividad 2). `main.go`, el
router de `cmd/`, los flags cortos y largos, la ayuda en dos niveles y
`version`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| draco no se podía ejecutar | Nada de lo hecho en la fase 2 se podía probar contra una instancia real |
| El parser de gtt adivinaba si un flag llevaba valor e ignoraba los desconocidos | `--isue OPS-7` corría sin issue; `--dry-run scan` se tragaba `scan` como valor |
| gtt escribía directo a la consola, siempre con colores | Sin tests de la salida, y códigos de color en pipes y logs de CI |

## 3. Diseño técnico

- **`main.go`** solo crea la app y sale con su código:
  `cmd.New(os.Stdout, os.Stderr).Run(os.Args[1:])`.
- **Router de mapa**, como en gtt y sin Cobra: `register(&command{run, help,
  summary}, nombres…)` con alias. La ayuda general lista **solo los comandos
  que existen** (hoy `version`), con sus alias; cada actividad suma el suyo.
  `TestEveryCommandHasHelpAndSummary` exige ayuda y resumen.
- **Flags declarados por comando** (`[]flag` con largo, corto y si lleva
  valor). `parseFlags` acepta `--issue X`, `--issue=X`, `-i X`, `-i=X` y
  `--`; conserva los argumentos sueltos (para `project add <nombre>`); un flag
  desconocido, uno sin valor o un valor en un flag que no lo lleva son error.
  Un flag sin valor no consume el argumento siguiente. Mejora sobre
  `parseFlagsWithShorts` de gtt, cuyos casos se portaron a los tests con datos
  genéricos.
- **Ayuda en dos niveles:** `draco help` / `-h` / `--help`, y `draco help
  <cmd>` / `draco <cmd> -h`. `-h` se atiende antes de ejecutar: nunca corre el
  comando; después de `--` es un argumento.
- **Códigos de salida:** 0 bien, 1 error, 2 uso incorrecto (comando o flag
  desconocido), con la sugerencia `ver draco help <cmd>`.
- **Salida testeable:** todo va a `App.Out`/`App.Err`; el color solo en una
  terminal real y sin `NO_COLOR`.
- **`version`** por ldflags (`internal/build.Version`, `dev` por defecto);
  también `--version` y `-v`.
- `draco` sin argumentos muestra la ayuda; en una terminal interactiva abrirá
  el modo visual en la fase 4.
- `make run args="…"` corre draco con `go run .`, sin compilar un binario
  (regla del proyecto para cuando exista el instalador).

## 4. Archivos

- `main.go`, `internal/build/version.go` (nuevos).
- `cmd/app.go`, `cmd/flags.go`, `cmd/help.go` y sus tests (nuevos).
- `Makefile` — `run`.
- `CLAUDE.md` — comandos, arquitectura y reglas del modo comando.
- `README.md` — estado y `make run`.
- `docs/roadmap.md` — actividad 2 de la fase 3.

## 5. Cómo verificar

```bash
make run args="help"
make run args="help version"
make run args="version --x"; echo $?     # 2
make test pkg=./cmd v=1
```
