#!/usr/bin/env bash
# Read-only checks before closing an activity. Run from the repo root.
set -u

ok()   { printf '  \033[32m✓\033[0m %s\n' "$1"; }
warn() { printf '  \033[33m⚠\033[0m %s\n' "$1"; warnings=$((warnings + 1)); }
fail() { printf '  \033[31m✗\033[0m %s\n' "$1"; failures=$((failures + 1)); }
section() { printf '\n\033[1m%s\033[0m\n' "$1"; }

warnings=0
failures=0

if [ ! -f go.mod ] || ! grep -q 'draco-hq' go.mod; then
	echo "Ejecuta el script desde la raíz de draco-hq." >&2
	exit 2
fi

base=develop
git rev-parse -q --verify origin/develop >/dev/null && base=origin/develop

section "1. Rama"
branch=$(git branch --show-current)
case "$branch" in
	main | develop | "") fail "estás en '${branch:-HEAD separado}': las actividades van en feature/<tema>, fix/<tema> o docs/<tema>" ;;
	feature/* | fix/* | docs/*) ok "rama $branch" ;;
	*) warn "rama '$branch' sin prefijo feature/, fix/ o docs/" ;;
esac

upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)
case "$upstream" in
	"") ok "sin upstream (el usuario lo fija con su primer push -u)" ;;
	origin/main | origin/develop) fail "la rama sigue a $upstream: un git push iría ahí. Corre: git branch --unset-upstream" ;;
	*) ok "upstream $upstream" ;;
esac

if git merge-base --is-ancestor "$base" HEAD; then
	ok "nace de $base actualizada"
else
	warn "$base tiene commits que esta rama no incluye: actualiza la rama antes del PR"
fi

commits=$(git rev-list --count "$base"..HEAD)
case "$commits" in
	0) ok "sin commits todavía" ;;
	1) ok "1 commit sobre $base" ;;
	*) warn "$commits commits sobre $base: la regla es uno por actividad" ;;
esac

if [ -n "$(git status --porcelain)" ]; then
	ok "cambios sin commitear: $(git status --porcelain | wc -l | tr -d ' ') archivo(s)"
fi

if [ -z "$(git config user.email)" ]; then
	fail "git no tiene user.email configurado"
fi

section "2. Go"
if out=$(go build ./... 2>&1); then ok "go build"; else fail "go build:"; echo "$out" | sed 's/^/      /'; fi
if out=$(go vet ./... 2>&1); then ok "go vet"; else fail "go vet:"; echo "$out" | sed 's/^/      /'; fi

unformatted=$(gofmt -l $(git ls-files -co --exclude-standard '*.go'))
if [ -z "$unformatted" ]; then ok "gofmt"; else fail "gofmt: $(echo "$unformatted" | tr '\n' ' ')"; fi

if out=$(go test ./... 2>&1); then
	ok "go test ($(echo "$out" | grep -c '^ok') paquetes)"
else
	fail "go test:"
	echo "$out" | grep -v '^ok' | sed 's/^/      /'
fi

section "3. Datos de organización (P-001, P-008)"
files=$(git ls-files -co --exclude-standard | grep -v '^\.claude/skills/cerrar-actividad/check\.sh$')
hits=$(echo "$files" | xargs grep -nIE \
	-e 'https?://[a-z0-9-]+\.atlassian\.net' \
	-e 'bitbucket\.org/[A-Za-z0-9_-]+' \
	-e '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.(com|net|org|ec|io)\b' \
	2>/dev/null | grep -vE 'empresa\.atlassian\.net|<empresa>|bitbucket\.org/mi-org|@example\.(com|org|test)|tu@correo\.com|noreply@anthropic\.com' || true)
if [ -z "$hits" ]; then ok "sin URLs de instancia, workspaces ni correos"; else warn "revisar:"; echo "$hits" | sed 's/^/      /'; fi

terms=.claude/terminos-privados.txt
if [ -f "$terms" ]; then
	found=$(grep -vE '^\s*(#|$)' "$terms" | while IFS= read -r term; do
		echo "$files" | xargs grep -nIiF -- "$term" 2>/dev/null
	done)
	if [ -z "$found" ]; then ok "ningún término de $terms"; else fail "términos privados en el repo:"; echo "$found" | sed 's/^/      /'; fi
else
	printf '  · sin %s (opcional, ignorado por git)\n' "$terms"
fi

section "4. Bitácora"
changed=$(git diff --name-only "$base" -- docs/bitacora/; git ls-files -o --exclude-standard docs/bitacora/)
entry=$(echo "$changed" | grep -E '^docs/bitacora/[0-9]{3}-[a-z0-9-]+\.md$' | sort -u)
if [ -z "$entry" ]; then
	fail "falta la entrada docs/bitacora/NNN-<tema>.md de esta actividad"
else
	for e in $entry; do
		if git cat-file -e "$base:$e" 2>/dev/null; then
			warn "$e ya existía en $base: solo se permiten notas aclaratorias"
		else
			ok "entrada $e"
		fi
		name=$(basename "$e")
		grep -q "($name)\|(\./$name)" docs/bitacora/README.md && ok "$name está en el índice" || fail "$name no está en docs/bitacora/README.md"
		for s in '## 1. Solicitud' '## 2. Motivación' '## 3. Diseño técnico' '## 4. Archivos' '## 5. Cómo verificar'; do
			grep -qF "$s" "$e" || fail "$name: falta la sección «${s#\#\# }»"
		done
	done
fi

section "Resultado"
if [ "$failures" -gt 0 ]; then
	printf '  \033[31m%d bloqueante(s)\033[0m, %d advertencia(s)\n' "$failures" "$warnings"
	exit 1
fi
printf '  \033[32mlisto para commit\033[0m (%d advertencia(s))\n' "$warnings"
