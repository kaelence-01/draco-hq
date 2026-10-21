#!/usr/bin/env bash
# Lleva develop a main: PR con merge commit (nunca squash ni rebase) y, ya
# fusionado, main y develop locales al día para el release. Lo corre el autor
# (necesita gh autenticado).
#   make promote
set -euo pipefail

remote=origin

fail() { echo "✗ $*" >&2; exit 1; }
ok() { echo "✓ $*"; }

command -v gh >/dev/null && gh auth status >/dev/null 2>&1 || fail "hace falta gh autenticado: gh auth login"
[ -z "$(git status --porcelain)" ] || fail "hay cambios sin commitear"
git fetch --quiet "$remote" main develop
if git rev-parse -q --verify refs/heads/develop >/dev/null; then
	git merge-base --is-ancestor develop "$remote/develop" ||
		fail "develop tiene commits que no están en $remote/develop: lo que llega a develop entra por PR"
fi
[ "$(git rev-list --count "$remote/main..$remote/develop")" -gt 0 ] || { ok "main ya tiene todo develop"; exit 0; }

pending=$(git log --no-merges --reverse --format='- %s' "$remote/main..$remote/develop")
echo "Llega a main:"
echo "$pending"

pr=$(gh pr list --base main --head develop --state open --json number --jq '.[0].number // empty')
if [ -n "$pr" ]; then
	ok "PR #$pr de develop a main ya abierto"
else
	url=$(gh pr create --base main --head develop --title "Integrar develop en main" --body "$(printf 'Actividades:\n\n%s\n' "$pending")")
	pr=${url##*/}
	ok "PR #$pr creado"
fi
url=$(gh pr view "$pr" --json url --jq .url)

read -r -p "¿Fusionar el PR #$pr con merge commit? [s/N]: " answer
[[ "$answer" =~ ^[sSyY] ]] || { echo "Sin fusionar: el PR #$pr queda abierto en $url"; exit 0; }

# No --delete-branch: the head is develop.
gh pr merge "$pr" --merge
ok "PR #$pr fusionado con merge commit"

git fetch --quiet "$remote" main develop
for b in main develop; do
	if [ "$(git branch --show-current)" = "$b" ]; then
		git merge --ff-only --quiet "$remote/$b"
	else
		git fetch --quiet "$remote" "$b:$b"
	fi
done
git switch --quiet main
ok "main y develop al día; sigues en main"
echo "Siguiente: make release bump=patch y, al terminar, git switch develop"
