# 025 — Proceso de release

**Fecha:** 2026-10-19
**Rama:** feature/release → develop
**Fase del roadmap:** 3

## 1. Solicitud

Actividad 7b de la fase 3: objetivos de release del Makefile, `checksums.txt`,
guía de release y primer release `v0.1.0` (sin marcar como pre-release, con
arm64; decisiones de la bitácora 024).

## 2. Motivación

| Problema | Daño real |
|---|---|
| Sin release no hay binario | Nadie puede usar draco sin Go ni el repo; el instalador y `draco update` (024) no tienen nada que descargar |
| El `make release` de gtt hacía `git add -A`, commit y push de la rama actual | Un archivo suelto o una rama equivocada terminaba publicado en el release |
| Las plataformas estaban en el Makefile y en el actualizador por separado | Una plataforma en uno solo sería un binario que nadie puede actualizar, o un 404 al actualizar |

## 3. Diseño técnico

- **`make release-build ver=vX.Y.Z`**: compila `PLATFORMS` (linux y darwin
  amd64/arm64, windows amd64) con `CGO_ENABLED=0 -trimpath -ldflags "-s -w -X
  …build.Version=<ver>"` en `dist/` (ignorada por git) y escribe
  `checksums.txt` con `sha256sum` o `shasum -a 256`, formato que lee
  `updater`. Solo compila: sirve para probar sin publicar.
- **`make release bump=patch|minor|major` o `ver=vX.Y.Z`** llama a
  `scripts/release.sh` (en un script y no en el Makefile para que se lea y se
  revise):
  - **guardas**: `gh` autenticado, rama `main`, árbol limpio, `main` igual a
    `origin/main`;
  - **versión**: desde el último tag (sin tags, `v0.0.0`); debe ser mayor y
    el tag no debe existir ni en local ni en `origin`;
  - **antes de publicar**: `make lint test` y `release-build`, y una
    confirmación;
  - **publicar**: tag anotado, push solo del tag, y `gh release create
    --generate-notes --verify-tag` con los binarios y `checksums.txt`.
  - **Sin commits ni push de ramas** (la diferencia con gtt). Si el release
    falla tras subir el tag, el script imprime el comando para reintentar.
- **Una sola lista de plataformas en la práctica**: un test de `updater`
  compara `PLATFORMS` del Makefile con `updater.Platforms`.
- **`develop` → `main`** por PR con *merge commit* (guía), para que las dos
  ramas compartan historia; ningún objetivo del Makefile escribe en `main`.
- **README**: sección «Instalación» (descargar, verificar con
  `checksums.txt`, ejecutar; avisos de SmartScreen y de la cuarentena de
  macOS por binarios sin firmar) y estado: fase 3 terminada.

## 4. Archivos

- `Makefile` — `PLATFORMS`, `release-build`, `release`, ayuda.
- `scripts/release.sh` (nuevo).
- `internal/updater/platforms_test.go` (nuevo).
- `.gitignore` — `dist/`.
- `docs/release.md` (nuevo) — guía: pasos, qué verifica, cómo probar y qué
  hacer si falla.
- `README.md` — instalación, estado y estructura.
- `CLAUDE.md` — comandos de release y regla de `main`.
- `docs/roadmap.md` — 7b hecha.

## 5. Cómo verificar

```bash
make release-build ver=v0.1.0 && (cd dist && sha256sum -c checksums.txt)
grep -ac v0.1.0 dist/draco-linux-amd64      # la versión quedó dentro (1)
make release bump=minor                     # fuera de main: se detiene
make test pkg=./internal/updater run=Makefile v=1
```

Probado: los cinco binarios compilan (≈9–10 MB cada uno), `checksums.txt`
verifica y la versión queda incrustada; fuera de `main` el script se detiene
antes de tocar nada.

**Pendiente del autor (criterio de cierre de la fase 3):** PR `develop` →
`main`, `make release bump=minor` (`v0.1.0`), instalar desde el release y
actualizar a un `v0.1.1` siguiendo `docs/release.md`.

## 6. Pendientes

- Firmar los binarios (Authenticode, notarización de Apple) para evitar los
  avisos de SmartScreen y Gatekeeper.
- CI que corra `make ci` en cada PR y `release-build` en cada tag.
- Homebrew/Scoop (nombre del paquete pendiente en el roadmap).
