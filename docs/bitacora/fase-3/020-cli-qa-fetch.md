# 020 — `draco qa` y `draco f`

**Fecha:** 2026-10-19
**Rama:** feature/cli-qa-fetch → develop
**Fase del roadmap:** 3

## 1. Solicitud

Seguir con la fase 3: la actividad 4, el consolidado QA y la exportación desde
el modo comando.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los flujos `qa` y `fetch` no se podían ejecutar | Nadie podía usar el consolidado ni exportar con draco |
| El flujo QA buscaba ubicaciones solo por título | Con destino Markdown, el **primer** consolidado no se podía crear: la carpeta vacía no tiene ninguno con ese título |

## 3. Diseño técnico

- **`draco qa`** recorre el flujo sobre el perfil, con la salida de gtt:
  pregunta el sprint si falta `-s` (Enter = Kanban) y el módulo si falta
  `-m`; verifica la cuenta (`qa_email`, con las dos cuentas en el error);
  `[1/3]` tareas, `[2/3]` una línea por tarea con ✓/✗ y **?** cuando una
  consulta falla (con su aviso), `[3/3]` publica. Si el consolidado existe se
  actualiza conservando lo escrito a mano; si le faltan secciones, la
  confirmación pasa a `[s/N]`. `--dry-run` muestra el Markdown. El tracker
  debe tener la capacidad QA (`Bundle.QA()`): si no, el error lo dice.
- **`draco f`** busca, elige (la única sin preguntar; con varias, Enter =
  la primera), exporta y guarda: `-o`, si no el **diálogo nativo** (zenity,
  PowerShell también desde WSL, osascript), si no la carpeta actual. Fuera de
  una terminal no abre diálogo. Busca en todos los spaces si el perfil no
  fija uno (`AnyScope`).
- **Diálogo de guardar** portado de gtt (`cmd/savedialog.go`), con
  `LookPath` (P-005) y el nombre escapado dentro del script de PowerShell
  (`''`) y de osascript.
- **Corrección en el flujo QA:** `Locations` usa la búsqueda por título y, si
  no encuentra consolidados, las ubicaciones del destino (`DocStore.Locations`
  con el prefijo). En Confluence sigue igual en la práctica; en Markdown
  ofrece la carpeta. El doble de los tests de `qa` entró en pánico al
  llamarse un método que no implementaba: es la red de seguridad prevista.
- **Tests sin red ni sistema:** `now` fija la ventana Kanban; `saveDialog`
  hace fallar el test si se abre fuera de una terminal; consolidado Kanban
  publicado en una carpeta vacía, Sprint con `--dry-run`, otra cuenta,
  sprint no numérico, tracker sin QA; exportar el único documento, elegir
  entre varios, `-o`, sin documentos.

## 4. Archivos

- `cmd/qa.go`, `cmd/fetch.go`, `cmd/savedialog.go` y sus tests (nuevos).
- `cmd/generate_test.go` — entorno simulado con config y tracker propios.
- `internal/qa/qa.go`, `qa_test.go` — respaldo de ubicaciones.
- `CLAUDE.md`, `README.md`, `docs/roadmap.md`.

## 5. Cómo verificar

```bash
make test pkg=./cmd v=1
make run args="help qa"
make run args="qa -s <N> -m <MÓDULO> --dry-run"   # contra tu Jira, no publica
make run args="f -i <TAREA>"
```

## 6. Pendientes

- Prueba manual del autor contra Jira y Confluence (`--dry-run` primero).
