# 021 — `draco backlog scan`

**Fecha:** 2026-10-19
**Rama:** feature/cli-backlog → develop
**Fase del roadmap:** 3

## 1. Solicitud

Seguir con la fase 3, actividad 5: `draco backlog scan` con el mismo informe y
los mismos flags que `gtt backlog scan`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los detectores de `internal/backlog` estaban portados pero no había forma de correrlos | Quien migra de gtt pierde el backlog técnico; la deuda solo se detecta a ojo |
| El parser de draco exige valor en todo flag que lo declara | `-o` sin ruta (guardar en la carpeta por defecto), uso habitual en gtt, sería un error de uso |

## 3. Diseño técnico

- **Mismo comando que gtt:** flags `-p`, `-r`, `--path`, `--since`, `-m`,
  `-l`, `-o`, `--json`, `--deps`; mismos destinos (proyecto > carpeta actual),
  mismo informe en pantalla, mismo JSON (`backlog.Report`) y misma regla de
  `-o` (archivo con un repo, carpeta con varios). El reporte guardado conserva
  todos los hallazgos; `--limit` solo recorta la pantalla.
- **Sin perfil ni credenciales:** solo lee repos locales, así que usa
  `loadConfig` y no `profile.Open`.
- **`backlog` es el comando y `scan` su subcomando**, como en gtt. `draco
  backlog` solo muestra la ayuda; otro subcomando es error de uso (2).
- **Flags `optional`** en `parseFlags`: el flag toma el siguiente argumento
  solo si no empieza por `-`. Así `-o`, `-o r.json`, `-o=r.json` y
  `-o -r frontend` funcionan; los demás flags siguen exigiendo valor.
- **Errores de uso con código 2:** `--limit` inválido y `-o archivo.json` con
  varios repos (en gtt eran error común). Un repo que no se puede analizar no
  corta los demás: se informa y el comando termina con 1.
- **`--json` deja stdout limpio:** los fallos van a `App.Err`.
- **`scanRepo`** (variable del paquete, `backlog.Scan` por defecto) se
  reemplaza en los tests, que alimentan reportes fijos sin git.
- El reporte por defecto va a `~/.config/draco/backlog/`, con permisos `0600`:
  guarda rutas internas y mensajes de commit.
- Se omitió el enlace a la guía de usuario de gtt en la ayuda: draco todavía no
  tiene guía. Los ejemplos de la ayuda usan `pagos-api` y el módulo `Pagos`.

## 4. Archivos

- `cmd/backlog.go`, `cmd/backlog_test.go` (nuevos) — comando, ayuda y tests
  (los de gtt con datos genéricos, más los de punta a punta con el doble).
- `cmd/flags.go`, `cmd/flags_test.go` — flags con valor opcional.
- `CLAUDE.md` — flags opcionales y `scanRepo`.
- `README.md` — estado del modo comando.
- `docs/roadmap.md` — actividad 5 de la fase 3 hecha.

## 5. Cómo verificar

```bash
make test pkg=./cmd run=Backlog v=1
make run args="help backlog"
make run args="backlog scan --path . -l 3"
make run args="backlog scan --path . --json"
```

## 6. Pendientes

- Guía de usuario de draco (la ayuda de gtt enlazaba a la suya).
- Decidir si se porta la skill `backlog-tareas` (roadmap).
