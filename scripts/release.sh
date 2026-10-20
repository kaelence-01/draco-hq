#!/usr/bin/env bash
# Publica un release de draco: tag en main y binarios con checksums.txt en
# GitHub Releases. Lo corre el autor (necesita su llave SSH y gh autenticado).
#   make release bump=patch|minor|major
#   make release ver=vX.Y.Z
set -euo pipefail

bump=${1:-}
ver=${2:-}
remote=origin

fail() { echo "✗ $*" >&2; exit 1; }
ok() { echo "✓ $*"; }

command -v gh >/dev/null && gh auth status >/dev/null 2>&1 || fail "hace falta gh autenticado: gh auth login"
[ "$(git branch --show-current)" = main ] || fail "los releases salen de main: git switch main && git pull --ff-only"
[ -z "$(git status --porcelain)" ] || fail "hay cambios sin commitear"
git fetch --quiet --tags "$remote" main
[ "$(git rev-parse HEAD)" = "$(git rev-parse "$remote/main")" ] || fail "main no está igual a $remote/main: git pull --ff-only"
ok "main limpia e igual a $remote/main"

last=$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || echo v0.0.0)
if [ -z "$ver" ]; then
	IFS=. read -r major minor patch <<<"${last#v}"
	case "$bump" in
	patch) ver="v$major.$minor.$((patch + 1))" ;;
	minor) ver="v$major.$((minor + 1)).0" ;;
	major) ver="v$((major + 1)).0.0" ;;
	*) fail "indica bump=patch|minor|major o ver=vX.Y.Z" ;;
	esac
fi
[[ "$ver" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "versión inválida: $ver (vMAJOR.MINOR.PATCH)"
[ "$ver" != "$last" ] && [ "$(printf '%s\n%s\n' "$last" "$ver" | sort -V | tail -1)" = "$ver" ] ||
	fail "$ver no es mayor que la última versión ($last)"
! git ls-remote --exit-code --tags "$remote" "refs/tags/$ver" >/dev/null 2>&1 || fail "el tag $ver ya existe en $remote"
! git rev-parse -q --verify "refs/tags/$ver" >/dev/null || fail "el tag $ver ya existe en local: git tag -d $ver"
ok "versión $last → $ver"

make --no-print-directory lint test
make --no-print-directory release-build ver="$ver"

read -r -p "¿Publicar $ver en GitHub? [s/N]: " answer
[[ "$answer" =~ ^[sSyY] ]] || { echo "Cancelado: no se creó el tag."; exit 0; }

git tag -a "$ver" -m "draco $ver"
git push "$remote" "$ver"
gh release create "$ver" dist/draco-* dist/checksums.txt --title "draco $ver" --generate-notes --verify-tag ||
	fail "el tag $ver se subió pero el release no se creó. Reintenta: gh release create $ver dist/draco-* dist/checksums.txt --title \"draco $ver\" --generate-notes --verify-tag"
ok "release $ver publicado: $(gh release view "$ver" --json url --jq .url)"
