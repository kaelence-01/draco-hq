# 001 — Base del core: migración desde gtt

**Fecha:** 2026-10-19
**Rama:** feature/core → develop
**Fase del roadmap:** 2

> Entrada escrita después de la fusión, al crear la bitácora (actividad 003).

## 1. Solicitud

Crear un sucesor de gtt con una interfaz visual en la terminal, sin IA, que
conserve el modo comando. Empezar portando la lógica de gtt a un proyecto nuevo
y separar los flujos de la entrada y salida de terminal.

## 2. Motivación

| Problema | Daño real |
|---|---|
| En gtt la lógica de cada comando vive mezclada con `fmt.Print` y lecturas de stdin | Una TUI tendría que duplicar la lógica o reescribir los comandos |
| gtt tenía fijos en el código el proyecto, los estados y los campos de Jira del QA | El binario solo servía a una instancia de Jira; draco es open source |
| Tres valores se interpolaban en JQL sin escapar | Un valor con comillas rompía la consulta o cambiaba su significado |
| `gtt` se confundía con `git` al escribirlo | Errores de tipeo frecuentes |

gtt queda congelado y no se modifica; Go no permite importar `internal/` de otro
módulo, así que la lógica se **copia**.

## 3. Diseño técnico

- **Paquetes portados con sus tests:** `atlassian`, `document`, `git`,
  `backlog`, `config`. Rutas y mensajes dicen `draco`
  (`~/.config/draco/config.yaml`).
- **QA configurable:** `qa_jira` en la config (`config.QAJiraConfig`), con
  `Validate()` que lista todos los problemas a la vez. Los métodos QA de
  `atlassian` reciben un `QASchema`; el JQL se arma en funciones puras con
  tests.
- **Escapado:** la fecha de la ventana Kanban (dos consultas) y el `accountId`
  del asignado, que gtt interpolaba con `"%s"` o `%q`, pasan por
  `quoteLiteral`.
- **Patrón de flujo** preparar → revisar → publicar, con `internal/deploy` como
  referencia: `Resolve` combina petición y config; una `Session` expone un
  método por paso; Atlassian entra por una interfaz con doble en los tests; lo
  degradado viaja como `Warnings` o campos, nunca se imprime.
- **Datos de prueba** heredados de gtt (módulos, repos, workspace, IDs)
  reemplazados por ejemplos genéricos.
- `docs/security/patrones-seguros.md` reescrito con ejemplos genéricos.

## 4. Archivos

- `internal/atlassian/` — cliente HTTP, Jira v3, Confluence v1/v2, `QASchema`,
  JQL puro, `quoteLiteral`.
- `internal/config/` — config YAML + variables de entorno, `QAJiraConfig`.
- `internal/document/` — ADF del documento de despliegue y del consolidado QA,
  preservación de «A considerar».
- `internal/git/`, `internal/backlog/` — portados sin cambios de comportamiento.
- `internal/deploy/` — flujo del documento de despliegue, 21 tests.
- `docs/roadmap.md`, `docs/security/patrones-seguros.md`, `CLAUDE.md`.
- `.gitignore` — `CLAUDE.local.md`.

## 5. Cómo verificar

```bash
go build ./... && go vet ./... && gofmt -l internal && go test ./...
```

## 6. Pendientes

Los que lista el roadmap en la fase 2 (QA, exportar, proyectos, convenciones de
texto configurables, listado de commits).
