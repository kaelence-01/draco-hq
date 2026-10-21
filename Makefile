# Atajos de draco. `make` (o `make help`) lista todo; las opciones van como
# variables: make start name=fetch, make test pkg=./internal/qa run=Golden v=1.

SHELL := /bin/bash
.DEFAULT_GOAL := help

REMOTE ?= origin
BASE   ?= develop
type   ?= feature
pkg    ?= ./...
time   ?= 20s
BRANCH := $(shell git branch --show-current 2>/dev/null)
# Release platforms; updater.Platforms must list the same (a test checks it).
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
VERSION_PKG := github.com/kaelence-01/draco-hq/internal/build
GOFILES = $(shell git ls-files --cached --others --exclude-standard '*.go')

.PHONY: help sync start push pr finish prune build run test vet fmt lint golden fuzz check ci release-build promote release

##@ Git

sync: ## Trae origin y actualiza develop solo con fast-forward, estés en la rama que estés
	@git fetch $(REMOTE)
	@if [ "$(BRANCH)" = "$(BASE)" ]; then git pull --ff-only $(REMOTE) $(BASE); \
	else git fetch $(REMOTE) $(BASE):$(BASE) && echo "$(BASE) al día (sigues en $(BRANCH))"; fi

start: ## Nueva actividad desde develop al día: make start name=<tema> [type=feature|fix|docs]
	@[[ "$(name)" =~ ^[a-z0-9]+(-[a-z0-9]+)*$$ ]] || { echo "name=<tema> en minúsculas y con guiones (ej: name=salida-markdown)"; exit 1; }
	@[[ "$(type)" =~ ^(feature|fix|docs)$$ ]] || { echo "type debe ser feature, fix o docs"; exit 1; }
	@[ -z "$$(git status --porcelain)" ] || { echo "hay cambios sin commitear: guárdalos o descártalos antes"; exit 1; }
	@$(MAKE) --no-print-directory sync
	git switch -c $(type)/$(name) $(BASE)
	@git branch --unset-upstream 2>/dev/null || true

push: ## Sube la rama actual y fija su upstream; force=1 reescribe con --force-with-lease
	@case "$(BRANCH)" in main|$(BASE)|"") echo "no se sube '$(BRANCH)' directo: trabaja en feature/<tema>"; exit 1;; esac
	git push -u $(REMOTE) $(BRANCH) $(if $(force),--force-with-lease,)

pr: ## Crea el PR de la rama actual hacia develop (gh); sin gh, muestra el enlace
	@case "$(BRANCH)" in main|$(BASE)|"") echo "no hay PR desde '$(BRANCH)'"; exit 1;; esac
	@if command -v gh >/dev/null && gh auth status >/dev/null 2>&1; then \
	gh pr create --base $(BASE) --head $(BRANCH) --fill $(if $(web),--web,); \
	else url=$$(git remote get-url $(REMOTE) | sed -E 's#^git@([^:]+):#https://\1/#; s#\.git$$##'); \
	echo "$$url/compare/$(BASE)...$(BRANCH)?expand=1"; fi

finish: ## Tras fusionar el PR: vuelve a develop, la actualiza y borra las ramas ya fusionadas
	git switch $(BASE)
	@$(MAKE) --no-print-directory sync prune

prune: ## Borra las ramas locales ya fusionadas en develop (ni main, ni develop, ni la actual, ni las recién creadas)
	@tip=$$(git rev-parse $(BASE)); \
	for b in $$(git branch --merged $(BASE) --format='%(refname:short)'); do \
		case "$$b" in main|$(BASE)|$(BRANCH)) continue;; esac; \
		[ "$$(git rev-parse "$$b")" = "$$tip" ] && continue; \
		git branch -d "$$b"; \
	done

##@ Go

build: ## Compila todo
	go build ./...

run: ## Corre draco sin instalarlo: make run args="help g"
	go run . $(args)

test: ## Tests: make test [pkg=./internal/qa] [run=Golden] [v=1]
	go test $(if $(v),-v,) $(if $(run),-run '$(run)',) $(pkg)

vet: ## Análisis estático
	go vet ./...

fmt: ## Formatea el código Go
	gofmt -w $(GOFILES)

lint: ## gofmt sin diferencias + go vet
	@out=$$(gofmt -l $(GOFILES)); [ -z "$$out" ] || { echo "sin formatear (corre make fmt):"; echo "$$out"; exit 1; }
	go vet ./...

golden: ## Regenera los goldens tras un cambio intencional: make golden [pkg=./internal/document]
	DRACO_UPDATE_GOLDEN=1 go test $(pkg) >/dev/null
	@git status --short -- '*/testdata/*'

fuzz: ## Fuzzing del conversor Markdown → ADF: make fuzz [time=1m]
	go test ./internal/provider/atlassian -run '^$$' -fuzz FuzzMarkdownToADF -fuzztime $(time)

##@ Actividad

check: ## Verificaciones antes del commit (skill cerrar-actividad)
	@bash .claude/skills/cerrar-actividad/check.sh

ci: lint test ## Lo que debe pasar en CI: lint + test

##@ Release

release-build: ## Binarios de todas las plataformas y checksums.txt en dist/: make release-build ver=v0.1.0
	@[[ "$(ver)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]] || { echo "ver=vMAJOR.MINOR.PATCH (ej: ver=v0.1.0)"; exit 1; }
	@rm -rf dist && mkdir dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; out=dist/draco-$$os-$$arch; [ $$os = windows ] && out=$$out.exe; \
		echo "  $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "-s -w -X $(VERSION_PKG).Version=$(ver)" -o $$out . || exit 1; \
	done
	@cd dist && { if command -v sha256sum >/dev/null; then sha256sum draco-*; else shasum -a 256 draco-*; fi; } > checksums.txt
	@echo "  dist/checksums.txt"

promote: ## Lleva develop a main (lo corre el autor): PR con merge commit, fusión confirmada, main y develop al día
	@bash scripts/promote.sh

release: ## Publica un release desde main (lo corre el autor): make release bump=patch|minor|major | ver=vX.Y.Z
	@bash scripts/release.sh "$(bump)" "$(ver)"

help: ## Muestra esta ayuda
	@awk 'BEGIN { FS = ":.*## " } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
	/^[a-z-]+:.*## / { printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo
	@echo "Opciones: name= type=feature|fix|docs  pkg= run= v=1  force=1  web=1  time=  args=  ver=  bump="
