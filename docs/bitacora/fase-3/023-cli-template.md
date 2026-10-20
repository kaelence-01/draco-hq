# 023 — `draco template`

**Fecha:** 2026-10-19
**Rama:** feature/cli-template → develop
**Fase del roadmap:** 3

## 1. Solicitud

Actividad 6b de la fase 3: `draco template init/list/validate/preview` sobre
el motor de plantillas (bitácora 015).

## 2. Motivación

| Problema | Daño real |
|---|---|
| Personalizar una plantilla exigía conocer los archivos embebidos y el orden de búsqueda | Nadie la personaliza, o edita una carpeta que draco no está usando |
| Un error en una plantilla solo aparecía al publicar | El documento real falla en el peor momento, con la tarea esperando |
| Una plantilla sin el título de una sección conservada no da error | Lo escrito a mano en «A considerar» se pierde sin aviso en la siguiente actualización |
| `draco g` y `draco qa` usaban la misma carpeta `template` del perfil para las dos plantillas | Una plantilla de despliegue en el perfil rompía el consolidado QA, y al revés |

## 3. Diseño técnico

- **`init <deploy|qa>`** copia los archivos de la plantilla incluida
  (`document.BuiltinFiles`) en `.draco/templates/<tipo>` del repo actual, en
  la carpeta del usuario (`--user`) o en otra (`--dir`). **Nunca sobrescribe**
  lo que ya existe sin `--force`. Una copia sin editar produce el mismo
  documento que la incluida (test).
- **`list [tipo]`** muestra las carpetas en el orden en que se buscan, cuál
  está en uso y cuáles quedan ignoradas porque gana una anterior. Solo mira si
  existen; no carga nada.
- **`validate [tipo]`** carga cada plantilla, la llena con datos de ejemplo
  (título y cuerpo) y avisa si al documento le falta el título de una sección
  de `preserve` (`Template.MissingPreserved`; los títulos dentro de bloques de
  código no cuentan). Sale con 1 si alguna no carga o no renderiza.
- **`preview [tipo]`** (por defecto `deploy`) muestra la plantilla en uso, el
  título y el Markdown con datos de ejemplo genéricos (`OPS-7`, `pagos-api`,
  `mi-org`, `example.test`). El checklist y los nombres del encabezado QA
  salen de la config, así la vista previa se parece al documento real.
- **Una sola regla de búsqueda:** `document.Dirs(kind, explícita, repo,
  usuario)` reemplaza las copias de `deploy` y `qa`; los comandos de
  `template` usan la misma, con el mismo repo (deploy: el backend del
  proyecto; qa: la carpeta actual).
- **Corrección: la carpeta de plantillas del perfil tiene una subcarpeta por
  tipo** (`deploy/`, `qa/`), como `~/.config/draco/templates`
  (`Profile.TemplateDir`). `--template` sigue siendo la carpeta de una sola
  plantilla, por eso en `list` y `validate` exige el tipo. La pregunta de
  `draco profile add` y el ejemplo de la arquitectura lo dicen.
- La comprobación de lo que un destino no soporta (prevista en la arquitectura
  para `validate`) queda pendiente: necesita que el conversor de cada
  adaptador informe lo que descarta.

## 4. Archivos

- `cmd/template.go`, `cmd/template_test.go` (nuevos).
- `cmd/generate.go`, `cmd/qa.go` — plantilla del perfil por tipo.
- `cmd/profile.go` — la pregunta de la carpeta de plantillas.
- `internal/document/template.go` — `Kinds`, `Dirs`, `BuiltinFiles`,
  `MissingPreserved`, con tests.
- `internal/deploy/deploy.go`, `internal/qa/qa.go` — usan `document.Dirs`.
- `internal/config/profile.go` — `Profile.TemplateDir`, con test.
- `docs/arquitectura/proveedores.md` — ejemplo de `template` en el perfil.
- `README.md`, `docs/roadmap.md` — estado.

## 5. Cómo verificar

```bash
make test pkg=./cmd run=Template v=1
make test pkg=./internal/document
make run args="template list"
make run args="template validate"
make run args="template preview qa"
```

## 6. Pendientes

- `validate`: avisar lo que el destino del perfil no soporta.
- Plantillas remotas (roadmap).
