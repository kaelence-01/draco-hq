# Cómo se publica una versión

Un release es un tag `vMAJOR.MINOR.PATCH` sobre `main` y un GitHub Release con
cinco binarios y su `checksums.txt`. `draco update` y el instalador solo
confían en ese archivo (P-006): un binario que no coincide no se instala.

| Binario | Sistema |
|---|---|
| `draco-linux-amd64`, `draco-linux-arm64` | Linux |
| `draco-darwin-amd64`, `draco-darwin-arm64` | macOS (Intel, Apple Silicon) |
| `draco-windows-amd64.exe` | Windows |

## Requisitos

- Tu llave SSH para `origin` y `gh` autenticado (`gh auth login`).
- GNU Make, Go 1.26, `sha256sum` o `shasum`.

## Pasos

1. **Llevar `develop` a `main`** con un PR. Fusiónalo con *merge commit*,
   nunca con *rebase and merge* ni *squash*: esos copian los commits con otros
   hashes, las ramas dejan de compartir historia y el siguiente PR choca en
   cada archivo (bitácora 026).

   ```bash
   gh pr create --base main --head develop --title "Release v0.1.0" --fill
   ```

2. **Publicar desde `main` al día:**

   ```bash
   git switch main && git pull --ff-only
   make release bump=patch       # o minor / major, o ver=vX.Y.Z
   ```

   Las fases 5 y 6 se publican como *patch* de `v0.2.x` (roadmap,
   «Decisiones tomadas»): `bump=minor` solo cuando el autor lo decida.

   `scripts/release.sh`:
   - se detiene si `gh` no está autenticado, si no estás en `main`, si hay
     cambios sin commitear o si `main` no es igual a `origin/main`;
   - calcula la versión desde el último tag (sin tags parte de `v0.0.0`:
     `bump=minor` da `v0.1.0`) y exige que sea mayor y que el tag no exista;
   - corre `make lint test` y `make release-build ver=<versión>`, que compila
     con `CGO_ENABLED=0 -trimpath` y la versión por ldflags en `dist/`;
   - pregunta antes de publicar; recién entonces crea el tag anotado, lo sube
     y crea el release con `gh release create --generate-notes --verify-tag`
     (las notas salen de los PR fusionados).

   No crea commits ni sube ramas.

3. **Volver a `develop`**: `git switch develop`.

## Verificar un release

El criterio de cierre de la fase 3 es instalar un release y actualizarlo:

1. Descarga el binario y `checksums.txt` del release, verifica y ejecútalo
   (README, «Instalación»). Debe ofrecer instalarse; acepta.
2. En una terminal nueva: `draco version` muestra la versión publicada.
3. Publica la siguiente (`make release bump=patch`; un tag nuevo sobre el
   mismo `main` sirve para probar).
4. `draco update --check` la anuncia y `draco update` la instala;
   `draco version` muestra la nueva.

## Si algo falla

- **Antes de la pregunta de publicar** no se creó nada: corrige y vuelve a
  correr.
- **El tag se subió pero el release no se creó**: el script imprime el
  `gh release create` para reintentar con los mismos archivos de `dist/`.
- **Un release con un binario malo**: no se reemplazan assets a mano;
  publica una versión nueva. Borrar un release ya descargado deja a la gente
  sin poder verificar lo que tiene.

## Lo que no se hace

- Marcar el release como *pre-release*: `releases/latest` lo excluye y
  `draco update` no lo vería. Una versión `v0.x` ya avisa que es temprana.
- Subir binarios compilados fuera de `make release-build`, o editar
  `checksums.txt`.
