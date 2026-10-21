#!/bin/sh
# Instala draco en Linux o macOS: descarga el binario del último release,
# verifica su SHA-256 contra checksums.txt y lo copia a ~/.local/bin.
#   curl -fsSL https://raw.githubusercontent.com/kaelence-01/draco-hq/main/scripts/install.sh | sh
# DRACO_VERSION=vX.Y.Z instala esa versión; DRACO_INSTALL_DIR cambia la carpeta.
set -eu

repo=kaelence-01/draco-hq
dir=${DRACO_INSTALL_DIR:-$HOME/.local/bin}

fail() { echo "✗ $*" >&2; exit 1; }
ok() { echo "✓ $*"; }

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "sistema no soportado: $(uname -s). En Windows usa scripts/install.ps1" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "arquitectura no soportada: $(uname -m)" ;;
esac
asset="draco-$os-$arch"

if [ -n "${DRACO_VERSION:-}" ]; then
	base="https://github.com/$repo/releases/download/$DRACO_VERSION"
else
	base="https://github.com/$repo/releases/latest/download"
fi

command -v curl >/dev/null || fail "hace falta curl"
if command -v sha256sum >/dev/null; then
	sum() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null; then
	sum() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
	fail "hace falta sha256sum o shasum para verificar la descarga"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "Descargando $asset..."
curl -fsSL "$base/$asset" -o "$tmp/$asset" || fail "no se pudo descargar $base/$asset"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || fail "no se pudo descargar checksums.txt"

want=$(awk -v a="$asset" '$2 == a { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || fail "checksums.txt no trae $asset"
[ "$(sum "$tmp/$asset")" = "$want" ] || fail "el SHA-256 no coincide con checksums.txt: no se instaló nada"
ok "SHA-256 verificado"

mkdir -p "$dir"
chmod 755 "$tmp/$asset"
mv -f "$tmp/$asset" "$dir/draco"
ok "Instalado en $dir/draco"

case ":$PATH:" in
*":$dir:"*) ;;
*)
	rc="$HOME/.bashrc"
	case "${SHELL:-}" in *zsh*) rc="$HOME/.zshrc" ;; esac
	line="export PATH=\"\$PATH:$dir\""
	if ! grep -qsF "$line" "$rc"; then
		printf '\n# draco\n%s\n' "$line" >>"$rc"
	fi
	ok "Carpeta agregada al PATH en $rc"
	echo "    Abre una terminal nueva o corre: source $rc"
	;;
esac

"$dir/draco" version
echo
echo "Siguiente paso: draco init (o draco, y Configuración)"
