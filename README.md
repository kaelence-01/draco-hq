# 🐉 draco

Herramienta de terminal para documentar despliegues: arma el **documento de
despliegue** de una tarea a partir del issue y de sus commits, publica el
**consolidado de QA** del sprint, **exporta** a texto la página de una tarea y
detecta **deuda técnica** en tus repos. Sin IA.

> **Estado: en desarrollo.** El núcleo (fase 2) y el modo comando (fase 3)
> están terminados: `help`, `version`, `update`, `init`, `project`, `profile` y
> `template` (configuración), `g` (documento de despliegue), `qa`
> (consolidado), `f` (exportar) y `backlog scan` (deuda técnica); los que
> publican, con `--dry-run`. El modo visual mínimo (fase 4) también: `draco`
> sin argumentos abre la pantalla principal, Proyectos y Generar, de la tarea a
> la publicación; QA, Exportar y Backlog llegan en la fase 5 (en curso). Los
> binarios se publican en Releases desde `v0.1.0`. Ver [`docs/roadmap.md`](docs/roadmap.md).

## Qué hace

| Función | Qué produce |
|---|---|
| **Documento de despliegue** | Una página con los archivos que cambian, agrupados por carpeta y enlazados al commit, y el checklist «A considerar». Si ya existe, la actualiza **sin perder lo que el equipo escribió a mano** |
| **Consolidado QA** | La tabla de controles de cada tarea del sprint (o de los últimos 10 días hábiles en modo Kanban). Una consulta que falla se muestra como «?», nunca como un control fallido |
| **Exportar** | El texto de la página que menciona una tarea, listo para guardar como `.txt` |
| **Backlog técnico** | Hotspots, archivos grandes, código sin tests, TODO/FIXME y dependencias vulnerables de tus repos locales, propuestos como tareas. Sin red |

## Instalación

Descarga el binario de tu sistema desde
[Releases](https://github.com/kaelence-01/draco-hq/releases/latest) junto con
`checksums.txt`, verifícalo y ejecútalo: la primera vez ofrece instalarse
(`~/.local/bin` o `%LOCALAPPDATA%\Programs\draco`) y agregarse al PATH.

```bash
# Linux (en macOS: draco-darwin-arm64 o -amd64 y shasum -a 256 -c)
curl -LO https://github.com/kaelence-01/draco-hq/releases/latest/download/draco-linux-amd64
curl -LO https://github.com/kaelence-01/draco-hq/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
chmod +x draco-linux-amd64 && ./draco-linux-amd64
draco init                           # credenciales, perfil y primer proyecto
```

En Windows descarga `draco-windows-amd64.exe` y ábrelo; compara antes su hash
(`Get-FileHash draco-windows-amd64.exe`) con `checksums.txt`. Los binarios no
están firmados: Windows SmartScreen pide «Más información → Ejecutar de todas
formas» y macOS puede pedir `xattr -d com.apple.quarantine <binario>`.

Después, `draco update` instala cada versión nueva verificando su SHA-256, y
draco avisa al usarlo cuando hay una.

## Dos formas de usarlo

**Modo comando**, para scripts, CI y quien ya conoce los comandos:

```bash
draco g -i OPS-7 -b a1b2c3d          # documento de despliegue
draco qa -s 12 -m Pagos              # consolidado QA del sprint 12
draco f -i OPS-7                     # exportar la página de OPS-7
draco backlog scan                   # deuda técnica
```

**Modo visual**: `draco` sin argumentos abre una interfaz dentro de la terminal,
con menús, listas para elegir commits, vista previa del documento, búsqueda de
ubicación y una barra de comandos con autocompletado (fases 4 a 6).

## No depende de una plataforma

Cada **perfil** combina un tracker, un destino de documentos y una plantilla:

```yaml
# ~/.config/draco/config.yaml
default_profile: equipo
profiles:
  equipo:
    tracker: { provider: jira }
    docs:    { provider: confluence, space: DEV }
  externo:
    tracker: { provider: jira }
    docs:    { provider: markdown, path: ./docs/despliegues }
```

Hoy: **Jira**, **Confluence** y **Markdown local** (archivos `.md` en una
carpeta), más git local. La arquitectura admite GitHub, GitLab, Azure DevOps,
Linear o Notion como nuevos adaptadores
([`docs/arquitectura/proveedores.md`](docs/arquitectura/proveedores.md)).

La estructura de los documentos sale de **plantillas** editables: un equipo
puede cambiar el checklist, el título o el documento completo sin recompilar,
desde `.draco/templates/` en su repo o `~/.config/draco/templates/`.

## Desarrollo

Requisitos: Go 1.26 y GNU Make.

```bash
make                     # lista todos los atajos
make run args="help"     # draco sin instalarlo (go run .)
make test                # tests (sin red: dobles, servidores simulados y repos temporales)
make lint                # gofmt + go vet
make test pkg=./internal/qa run=Kanban v=1
make golden              # regenera los goldens tras un cambio intencional
```

Estructura:

```
main.go, cmd/            modo comando
internal/
  deploy/ qa/ fetch/     flujos, sin entrada ni salida de terminal
  backlog/               detectores de deuda técnica
  document/              documentos en Markdown desde plantillas
  provider/              modelo neutro y puertos
    atlassian/ markdown/ git/   adaptadores
  profile/               arma los adaptadores según el perfil
  config/                config.yaml y variables de entorno
  installer/ updater/    instalación en el primer uso y draco update
scripts/release.sh       publica un release (make release)
docs/
  roadmap.md             fases, estado y decisiones
  release.md             cómo se publica una versión
  arquitectura/          diseño de proveedores y plantillas
  bitacora/fase-N/       una entrada por cambio, con su porqué
  security/              patrones de seguridad
```

### Cómo se trabaja

- Cada cambio nace de `develop` en `feature/<tema>` (`make start name=<tema>`),
  es **un solo commit** y llega a `develop` por PR (`make push`, `make pr`).
  `main` solo recibe lo integrado.
- Cada cambio lleva su entrada de bitácora en `docs/bitacora/fase-N/`;
  `make check` verifica eso, el formato y los tests antes del commit.
- Ningún dato de una organización en el código, los tests ni la documentación:
  URLs de instancia, claves de proyecto, IDs y nombres van a la configuración
  ([`docs/security/patrones-seguros.md`](docs/security/patrones-seguros.md)).

## Origen

draco es el sucesor de **gtt** ([`geomark27/deploy-doc`](https://github.com/geomark27/deploy-doc)),
un CLI que queda congelado. Su lógica se portó aquí, se separó de la terminal y
se desacopló de Atlassian.

## Licencia

[MIT](LICENSE)
