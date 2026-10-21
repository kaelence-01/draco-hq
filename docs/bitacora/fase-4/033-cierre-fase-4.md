# 033 — Cierre de la fase 4

**Fecha:** 2026-10-20
**Rama:** docs/cierre-fase-4 → develop
**Fase del roadmap:** 4

## 1. Solicitud

Hacer la actividad 5 de la fase 4: el cierre de la fase y el release
`v0.2.0`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `draco help` no decía que `draco` sin argumentos abre el modo visual | Quien instala el binario solo ve la ayuda y no sabe que existe la TUI, la novedad de `v0.2.0` |
| El roadmap, el README y el diseño decían «en curso» | Con la fase terminada, el estado publicado quedaba viejo |

## 3. Diseño técnico

- **Ayuda general:** la primera línea de «Uso:» es `draco` y dice «Modo
  visual: menús, listas y formularios (en una terminal)». `TestGeneralHelp`
  lo exige.
- **Fase 4 cerrada** en el roadmap (estado y sección), con su párrafo de
  cierre. El README y `docs/arquitectura/tui.md` dicen qué está hecho y que
  QA, Exportar y Backlog llegan en la fase 5.
- **Lo que entregó la fase** (bitácoras 028 a 032):
  - el diseño y el plan;
  - la base de la TUI: marco, pila de pantallas, ruta, panel y teclas que
    una pantalla puede capturar;
  - Proyectos con su formulario;
  - Generar completo: tarea, commits, progreso con cancelación, vista
    previa, ubicación y publicación;
  - `deploy.ProjectSides` y `ChangeSource.CurrentBranch`;
  - las normas obligatorias del modo visual.
- **Release `v0.2.0`** (lo publica el autor, que tiene la llave SSH y `gh`):
  `develop` → `main` por PR con *merge commit*, y `make release bump=minor`
  desde `main` al día (`docs/release.md`). Es *minor* porque suma una función
  (el modo visual) sin romper el modo comando.
- **Verificación del criterio de cierre**, después del release:
  1. con `v0.1.x` instalado, `draco update --check` anuncia `v0.2.0`;
  2. `draco update` la instala y `draco version` responde `v0.2.0`;
  3. en una terminal nueva, `draco` sin argumentos abre la TUI desde el
     binario instalado, y `draco help` muestra la línea del modo visual.

## 4. Archivos

- `cmd/help.go`, `cmd/app_test.go`: la línea del modo visual en la ayuda.
- `docs/roadmap.md`: fase 4 cerrada y su cierre.
- `README.md`, `docs/arquitectura/tui.md`: el estado.
- `docs/bitacora/fase-4/033-cierre-fase-4.md`, índice.

## 5. Cómo verificar

```bash
make check
make run args="help" | head -5      # la línea «draco  Modo visual: …»
# después del merge a main (lo corre el autor):
git switch main && git pull --ff-only
make release bump=minor             # v0.2.0
draco update --check && draco update && draco version
draco                               # la TUI, desde el binario instalado
```

## 6. Pendientes

- Release `v0.2.0` y su verificación (arriba), a cargo del autor.
- Pruebas manuales de las bitácoras 029 a 032: Windows Terminal y Linux/WSL,
  y Generar contra Jira y Confluence reales. Un error ahí se corrige con un
  `fix/` y un patch (`v0.2.1`).
- Fase 5: Backlog (tabla filtrable), QA (progreso en vivo) y Exportar, con
  plan antes de escribir código, como en la fase 4.
