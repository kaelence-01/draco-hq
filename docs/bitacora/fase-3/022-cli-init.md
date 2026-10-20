# 022 — `draco init`, `draco project` y `draco profile`

**Fecha:** 2026-10-19
**Rama:** feature/cli-init → develop
**Fase del roadmap:** 3

## 1. Solicitud

Actividad 6 de la fase 3, dividida en dos a pedido del autor: **6a** (esta)
con `init`, `project` y `profile`; **6b** con `draco template`. Leer el token
sin mostrarlo, agregando `golang.org/x/term` (aprobado por el autor).

## 2. Motivación

| Problema | Daño real |
|---|---|
| draco solo se configuraba editando YAML a mano | Un error de sangría o un campo mal escrito y ningún comando funciona; nadie lo adopta sin leer el código |
| Los perfiles existían en `config` pero ningún comando los creaba | Para usar el destino Markdown había que conocer el formato interno |
| gtt mostraba el token en pantalla al escribirlo | El token queda en el historial visual, en grabaciones y a la vista de quien esté cerca |
| `LoadLocal` mezcla las variables `ATLASSIAN_*` con el archivo | Un comando que leyera así y guardara escribiría en el disco un token que solo vivía en el entorno |

## 3. Diseño técnico

- **`config.LoadFile`**: lee solo el archivo (vacío si no existe). Todo
  comando que guarda parte de ahí; `LoadLocal` queda para los que solo leen.
  `ValidName` se exporta para validar el nombre antes de las preguntas.
- **`draco init`** (como gtt): email, token y URL (debe empezar con
  `https://`; se quita la `/` final). Si no hay perfiles, crea uno (nombre
  `equipo` por defecto) con destino Confluence + space o carpeta Markdown, y
  lo deja por defecto. Guarda, valida las credenciales con `Tracker.Me` a
  través de `profile.Open` y ofrece corregir el email si no es el de la cuenta
  (el token funciona pero las búsquedas por autor fallarían). Termina
  ofreciendo un proyecto. Con una config existente pide confirmación y
  conserva proyectos, perfiles y ajustes; Enter conserva cada valor actual,
  también el token.
- **Token oculto**: `App.askSecret` usa `term.ReadPassword` cuando la entrada
  es una terminal; en tests y pipes lee una línea. El token nunca se imprime.
- **Preguntas**: `askValue` (Enter toma el valor por defecto; un dato
  obligatorio se repite) y `askLine`, que devuelve `errEndOfInput` si la
  entrada termina, para que un asistente sin respuestas no quede en un bucle.
- **`draco project list|add|default|remove`** (alias `ls`, `rm`), como gtt.
  `add [nombre]` también pregunta host y organización del VCS y el space del
  proyecto; el nombre del repo propone la carpeta de la ruta. Reemplazar un
  proyecto parte del existente: se conserva lo que el asistente no pregunta
  (`deploy_checklist`, `backlog`).
- **`draco profile list|add|default|remove`** (nuevo). El tracker es Jira, el
  único disponible. **El primer perfil no rompe lo que funcionaba:** una config
  sin perfiles se lee como el perfil implícito `atlassian`; al agregar otro, ese
  también se guarda y queda por defecto, así `draco g` sigue yendo a
  Confluence.
- **Subcomandos**: `App.subcommand` resuelve `project`, `profile` y ahora
  también `backlog`; sin argumentos muestran la ayuda y uno desconocido es
  error de uso (2).
- **Limitación conocida (igual que gtt)**: `config.Save` reescribe el YAML y
  se pierden los comentarios escritos a mano. Lo dicen las ayudas.

## 4. Archivos

- `cmd/init.go`, `cmd/project.go`, `cmd/profile.go` y sus tests (nuevos).
- `cmd/app.go` — `askLine`, `askValue`, `askSecret`, `errEndOfInput`.
- `cmd/backlog.go` — usa `App.subcommand`.
- `internal/config/config.go` — `LoadFile`; `profile.go` — `ValidName`.
- `go.mod`, `go.sum` — `golang.org/x/term`.
- `CLAUDE.md` — dependencias y regla de `LoadFile`.
- `README.md` — estado del modo comando.
- `docs/roadmap.md` — actividad 6 dividida en 6a (hecha) y 6b.

## 5. Cómo verificar

```bash
make test pkg=./cmd run='Init|Project|Profile' v=1
# en un HOME temporal, para no tocar la config real:
HOME=$(mktemp -d) go run . init
HOME=$(mktemp -d) go run . profile list
```

Prueba manual del autor: `draco init` contra su sitio de Atlassian (token
oculto y validación de la cuenta).

## 6. Pendientes

- 6b: `draco template init/list/validate/preview`.
- Conservar los comentarios del YAML al guardar (exigiría editar con
  `yaml.Node`); por ahora lo avisan las ayudas.
- Gestor de secretos del sistema para el token (decisión pendiente del roadmap).
