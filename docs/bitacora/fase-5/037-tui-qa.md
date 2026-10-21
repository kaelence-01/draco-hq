# 037 — QA en la TUI (1): cuenta, tareas y evaluación en vivo

**Fecha:** 2026-10-20
**Rama:** feature/tui-qa → develop
**Fase del roadmap:** 5

## 1. Solicitud

Seguir con la siguiente actividad de la fase 5: QA (1), actividad 4a del
plan de la bitácora 034.

## 2. Motivación

| Problema | Daño real |
|---|---|
| «Consolidado QA» estaba atenuado en el menú | QA solo veía la evaluación de las tareas como líneas de texto al correr `draco qa`, sin poder recorrerlas ni leer las observaciones de una |
| Evaluar todas las tareas lleva varias consultas por tarea | Sin progreso por tarea, la pantalla quedaría quieta durante la parte más larga del consolidado |

## 3. Diseño técnico

- **Formulario** (`internal/tui/qa.go`): sprint (vacío: Kanban; si lo hay,
  número positivo) y módulo. Al enviar:
  1. abre el perfil de la sesión y pide `Bundle.QA()`; `ErrNoQA` se explica
     («usa un perfil con tracker jira»);
  2. `qa.Resolve` con `Deps.Now` (el `now` de `cmd`) y la plantilla del
     perfil, como `draco qa`; el módulo que falta en modo Sprint lo explica
     `Resolve`.

  El perfil se abre antes de `Resolve` porque la plantilla sale de él; el
  alcance es el del perfil (la TUI no tiene `--space`).
- **Evaluar** (progreso en vivo):
  - `CheckAccount`, `Collect`, un `EvaluateTask(i)` por tarea y
    `FindExisting`, cada uno en un `tea.Cmd`, en orden. Cada fila aparece al
    terminar su tarea, con las marcas `✓`/`✗` y `?` para
    `DeployDocUnknown`; los avisos de cada tarea van debajo de la tabla.
  - `ErrNotQAAccount` muestra la cuenta autenticada, el `qa_email` del
    config y la ruta del config, como el comando.
  - `esc` cancela entre tareas.
  - **Copia propia de las filas:** la pantalla copia `Session.Tasks` al
    terminar `Collect` y, de cada evaluación, solo la fila evaluada. Así no
    lee la `Session` mientras otro paso la escribe (norma 5).
  - Al terminar: `↑↓` recorre las filas; debajo van la tarea de revisión y
    las observaciones de la elegida, qué hará publicar (crear o actualizar
    conservando lo escrito a mano) y el comando equivalente.
  - La vista previa y la publicación son la actividad 4b: hasta entonces la
    pantalla termina en el comando para publicar.
- **Compacta para 24 líneas:** la cuenta y el recuento de tareas van en una
  línea, y el comando equivalente en otra. En dos líneas cada uno, el
  comando quedaba fuera de la vista con dos tareas y un aviso.
- **Menú:** «Consolidado QA» abre la pantalla. Solo Configuración sigue
  atenuada (fase 6): los tests de la principal la usan ahora como ejemplo de
  función sin pantalla.

## 4. Archivos

- `internal/tui/qa.go`, `qa_test.go` — nuevos: formulario, evaluación y sus
  tests (Kanban de punta a punta, sprint con consolidado existente, módulo
  obligatorio, otra cuenta, perfil sin QA, filas en vivo con `esc` que
  cancela, 80 columnas).
- `internal/tui/tui.go`, `cmd/visual.go` — `Deps.Now`.
- `internal/tui/home.go`, `app_test.go` — «Consolidado QA» con pantalla.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  detalle.
- `docs/bitacora/fase-5/037-tui-qa.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run=QA v=1
make check
make run            # Consolidado QA con la cuenta de QA contra Jira
```

## 6. Pendientes

- Actividad 4b: vista previa, ubicación y publicar, compartidos con Generar.
- Pruebas manuales del autor: la evaluación contra su Jira, en modo Sprint y
  Kanban, en Windows Terminal y Linux/WSL.
