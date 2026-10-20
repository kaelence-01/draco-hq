# 024 — Instalador, actualizador y `draco update`

**Fecha:** 2026-10-19
**Rama:** feature/updater → develop
**Fase del roadmap:** 3

## 1. Solicitud

Actividad 7 de la fase 3, dividida en dos: **7a** (esta) con el instalador, el
actualizador y `draco update`; **7b** con el proceso de release. Decisiones
del autor: publicar también arm64, primera versión `v0.1.0` sin marcar como
pre-release, e instalación en el primer uso como gtt.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Un binario descargado no queda en el PATH | Hay que copiarlo y editar el PATH a mano; en Windows casi nadie sabe cómo |
| Sin actualizador, cada versión nueva se reinstala a mano | Los equipos se quedan en versiones viejas con errores ya corregidos |
| Descargar y ejecutar un binario sin verificarlo | Un asset alterado o corrupto se instalaría y correría con los permisos del usuario (P-006) |
| El actualizador de gtt no tenía tests | Un error en la verificación o el reemplazo solo se vería en la máquina de un usuario |

## 3. Diseño técnico

- **`internal/updater`**:
  - `Client` con las URLs de la API y de descargas como campos: los tests
    usan un GitHub simulado (`httptest`), nunca la red.
  - `Latest` lee `releases/latest`, que excluye las pre-releases (por eso
    `v0.1.0` no se marcará como tal). Un 404 es `ErrNoReleases`: el repo
    todavía no publicó ninguna versión.
  - `Fetch` baja `checksums.txt` y el binario de la plataforma, y lo devuelve
    **solo si su SHA-256 coincide**. Una plataforma sin binario publicado es
    error antes de descargar, y una descarga de más de 200 MB se corta.
  - `Replace` escribe junto al ejecutable y renombra, así una falla nunca
    deja medio binario. En Windows el `.exe` en uso se aparta como `.old` y
    `CleanOld` lo borra en el siguiente arranque.
  - `IsNewer` compara `vMAJOR.MINOR.PATCH`; `dev` nunca es más nuevo ni más
    viejo, así una versión de desarrollo jamás se reemplaza.
  - `Notifier`: el aviso sale de `~/.config/draco/version_check.json`, sin
    red. La caché se renueva como mucho una vez al día y una consulta
    fallida no borra lo que ya se sabía.
- **Plataformas** (`updater.Platforms`): linux y darwin en amd64 y arm64, y
  windows amd64. Los binarios se llaman `draco-<os>-<arch>[.exe]`.
- **`internal/installer`**:
  - `IsInstalled(exe, dir)` acepta el binario si corre desde la carpeta de
    instalación o si es un build de `go run`/`go test` (`/go-build`).
  - `Install` copia con escritura y renombre, y agrega la carpeta al PATH
    solo si no está. La comparación es por entrada completa: `bin-old` no
    cuenta como `bin`.
  - En Unix escribe una sola vez la línea en `.zshrc` o `.bashrc`.
  - En Windows escribe en el registro con `golang.org/x/sys`, conservando el
    tipo `REG_EXPAND_SZ` (decisión heredada de gtt).
- **`App.Start`** (lo llama `main.go`) es `Run` más lo que solo tiene sentido
  con una persona en una terminal (`App.tty`: entrada y salida son
  terminales):
  - Fuera de su carpeta, ofrece instalarse. Sin argumentos (doble clic en
    Windows) termina ahí y espera Enter; con un comando, el comando corre
    igual, se instale o no.
  - El aviso de versión nueva sale después del comando. No aparece en pipes,
    en CI, en `update`, `version` ni `help`, ni con `DRACO_NO_UPDATE_CHECK`.
    Si no hay nada en caché, se espera la consulta como mucho 1,5 s.
- **`draco update [--check]`**: consulta, muestra versión actual y nueva,
  descarga, verifica y reemplaza. Guarda la respuesta en la caché para que el
  aviso desaparezca.
- **Mejoras sobre gtt**:
  - El aviso y el instalador viven en `cmd` y se prueban sobre `App`; en gtt
    estaban en `main.go`, sin tests.
  - Rechazar un binario alterado tiene test.
  - Con un comando, rechazar la instalación no impide usarlo.
  - El PATH no se toca si la carpeta ya está.

## 4. Archivos

- `internal/updater/updater.go`, `notify.go` y tests (nuevos).
- `internal/installer/installer.go`, `path_unix.go`, `path_windows.go` y tests
  (nuevos).
- `cmd/update.go`, `cmd/startup.go`, `cmd/update_test.go` (nuevos).
- `cmd/app.go` — `App.tty`.
- `main.go` — llama a `Start`.
- `go.mod`, `go.sum` — `golang.org/x/sys` directo.
- `CLAUDE.md`, `README.md`, `docs/roadmap.md` — arquitectura, dependencias y
  estado; la actividad 7 dividida en 7a y 7b.

## 5. Cómo verificar

```bash
make test pkg=./internal/updater v=1
make test pkg=./internal/installer v=1
make test pkg=./cmd run='Update|Start|ChecksUpdates' v=1
GOOS=windows go vet ./...     # el instalador de Windows compila
make run args="update --check"  # sin releases: «todavía no hay versiones publicadas»
```

La prueba de punta a punta (instalar un binario publicado y actualizarlo) es
el criterio de cierre de la 7b, con el primer release.

## 6. Pendientes

- 7b: objetivos de release del Makefile, `checksums.txt`, guía y `v0.1.0`.
- Windows arm64: no se publica todavía.
