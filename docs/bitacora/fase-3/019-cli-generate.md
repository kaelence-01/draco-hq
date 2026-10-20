# 019 — `draco g`: documento de despliegue desde el comando

**Fecha:** 2026-10-19
**Rama:** feature/cli-generate → develop
**Fase del roadmap:** 3

## 1. Solicitud

Seguir con la fase 3: `draco g` (actividad 3), el documento de despliegue de
punta a punta desde el modo comando.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El flujo `deploy` existía pero nadie lo podía ejecutar | Sin probarlo contra una instancia real, nada asegura que la fase 2 funciona fuera de los tests |
| gtt mezclaba en `generate.go` la lógica con la terminal | Portarlo tal cual habría duplicado lo que ya hace el flujo |

## 3. Diseño técnico

- **El comando recorre el flujo, no lo reimplementa:** `deploy.Resolve` →
  `profile.Open` → `Session` (`FetchIssue`, `FindExisting`, `ReadCommits`,
  `DraftCreate`/`DraftUpdate`, `Locations`, `Publish`). La salida y las
  preguntas son las de gtt: pasos `[1/4]`…`[4/4]`, «[1] Actualizar [2] Crear
  nuevo [3] Cancelar», ubicación por número y confirmación `[S/n]`.
- **Perfil:** la plantilla del perfil se usa si no hay `--template`; el space
  o la carpeta: `--space` > proyecto > perfil (lo completa `profile.Open`).
  Los errores del perfil se explican con qué hacer (`explainProfile`).
- **Secciones a conservar:** si el documento actual no tiene alguna de las
  que declara la plantilla, se avisa y se pregunta (`[s/N]`); si las tiene,
  se informa que se conservan.
- **`--dry-run` muestra el Markdown** (no el ADF, como gtt): legible e
  independiente del destino. Lee la tarea y los commits; no escribe ni
  enlaza nada (probado).
- **Fallos parciales:** un lado de commits que falla se marca con ✗ y se
  publica con el otro; un enlace que falla se avisa con qué hacer, porque el
  consolidado QA lo verifica.
- **`App`** suma la entrada (`In`, para preguntas), `ask`/`confirm` (Enter es
  la respuesta por defecto; al final de la entrada también) y un `context`
  que Ctrl+C cancela.
- **Tests sin red:** `loadConfig` y `openProfile` se reemplazan por una
  config vacía y un perfil con tracker y git simulados y el **adaptador
  Markdown real** en una carpeta temporal: crear, editar el archivo a mano y
  actualizar conservando la edición; `--dry-run`; cancelar; opción inválida;
  lado que falla; enlace que falla; tarea inexistente.
- Detalle: en Markdown la ubicación raíz se muestra «./ (raíz de la
  carpeta)».

## 4. Archivos

- `cmd/generate.go`, `cmd/generate_test.go` — nuevos.
- `cmd/app.go`, `cmd/app_test.go`, `main.go` — entrada, preguntas y
  `context`.
- `internal/provider/markdown/markdown.go` — título de la ubicación raíz.
- `README.md`, `docs/roadmap.md`.

## 5. Cómo verificar

```bash
make test pkg=./cmd run=Generate v=1
make run args="help g"
# contra tu instancia (lee Jira y git, no publica):
make run args="g -i <TAREA> -b <hash> -p <proyecto> --dry-run"
```

## 6. Pendientes

- **Prueba manual del autor contra Confluence** (con `--dry-run` primero y
  después publicando en un space de prueba). Es el criterio de cierre que no
  se puede automatizar.
