# 016 — Perfiles y cierre de la fase 2

**Fecha:** 2026-10-19
**Rama:** feature/perfiles → develop
**Fase del roadmap:** 2

## 1. Solicitud

Terminar la fase 2: la actividad 7 de la secuencia de proveedores, perfiles,
con el alcance recomendado (configuración con perfiles, migración en memoria,
fábrica de adaptadores y funciones puras de proyectos y perfiles); el
asistente `draco init` y el gestor de secretos quedan para después.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Ya existían dos destinos (Confluence y Markdown) pero nada elegía entre ellos | Cada llamador tendría que decidir qué adaptador construir, y el comando y la TUI lo harían distinto |
| Los flujos exigían un space de Confluence (`ErrNoSpace`) | Un perfil Markdown no podía funcionar aunque no use space |
| Los usuarios de hoy tienen un `config.yaml` sin perfiles | Si hubiera que migrarlo a mano, la actualización rompería a todos |

## 3. Diseño técnico

- **Config:** `profiles` (cada uno con `tracker`, `docs` y `template`) y
  `default_profile`. `config.Profile(nombre)` elige: el nombre, si no
  `default_profile`, si no el único perfil.
- **Migración en memoria:** sin `profiles`, la config se lee como el perfil
  `atlassian` (Jira + Confluence con `confluence_space_key`); no se reescribe
  el archivo. Probado leyendo un `config.yaml` de hoy desde YAML real.
- **`internal/profile`:** `Open(cfg, Options{Name, Scope, AnyScope})` arma
  tracker, destino y cambios (Jira, Confluence o Markdown local, git local),
  con un solo cliente de Atlassian para Jira y Confluence. Errores claros:
  proveedor desconocido (con la lista de válidos), credenciales que faltan
  (`ErrNoCredentials`), carpeta de Markdown que falta.
- **El space pasa del flujo al perfil:** `deploy`, `qa` y `fetch` ya no caen al
  `confluence_space_key` global ni devuelven `ErrNoSpace`; `Target.SpaceKey`
  es solo lo que pidió el usuario (flag o proyecto). `Open` completa con el
  del perfil y exige space solo para Confluence; `AnyScope` deja que exportar
  busque en todos los spaces, como antes.
- **QA como capacidad:** `Bundle.QA()` devuelve el `QATracker`, el error de
  validación de `qa_jira` (más útil que un genérico) o `ErrNoQA` si el
  tracker no tiene la capacidad.
- **Funciones puras** de proyectos y perfiles (agregar, quitar,
  predeterminado; quitar el predeterminado deja sin predeterminado), con
  validación del nombre; se guardan con `config.Save`. Las usarán `draco init`
  y los formularios de la TUI.
- **No se hizo, a propósito:** URL y credenciales por perfil (hoy son
  globales: nadie lo necesita todavía) y el gestor de secretos del sistema
  (requiere una dependencia).

### Cierre de la fase 2

Con esta actividad la fase 2 queda cerrada: el core portado de gtt, los cuatro
flujos sin E/S de terminal, puertos con tres adaptadores (Atlassian, Markdown
local, git), documentos en Markdown desde plantillas, preservación de lo escrito
a mano en el destino, perfiles, goldens, bitácora y Makefile. Lo que quedó
explícitamente para la fase 3 está en el roadmap (comando, `--template`,
`--profile`, `draco init`, diálogos de guardar, instalador, updater,
releases).

## 4. Archivos

- `internal/config/profile.go`, `profile_test.go` (nuevos); `config.go` —
  campos y fusión de `profiles`.
- `internal/profile/profile.go`, `profile_test.go` (nuevos).
- `internal/deploy/deploy.go`, `internal/qa/qa.go`, `internal/fetch/fetch.go`
  y sus tests — fuera el space global y `ErrNoSpace`.
- `docs/arquitectura/proveedores.md` (§8), `docs/roadmap.md` (fase 2 cerrada,
  fase 3 siguiente), `CLAUDE.md`.

## 5. Cómo verificar

```bash
make test pkg=./internal/profile v=1
make test pkg=./internal/config run=Profile v=1
make check
```

## 6. Pendientes

- Fase 3: `--profile` y `--space` en los comandos, `draco init` sobre las
  funciones puras.
- URL y credenciales por perfil; gestor de secretos del sistema.
