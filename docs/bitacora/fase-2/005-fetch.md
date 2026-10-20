# 005 — Flujo de exportar (internal/fetch)

**Fecha:** 2026-10-19
**Rama:** feature/fetch → develop
**Fase del roadmap:** 2

## 1. Solicitud

Seguir con la fase 2: portar `gtt fetch` (`cmd/fetch.go`) a un flujo sin E/S de
terminal, punto 1 del roadmap tras QA.

## 2. Motivación

| Problema | Daño real |
|---|---|
| En gtt la búsqueda, la elección de página, la conversión y la escritura del archivo viven en una función con prompts | La TUI no puede ofrecer una lista de páginas con vista previa sin duplicar la lógica |
| `sanitizeFilename` borraba las letras con tilde | «Validación» quedaba «Validacin» en el nombre del archivo, algo frecuente con títulos en español |
| Una clave escrita en minúsculas (`ops-7`) quedaba así en el encabezado y en el nombre del archivo | Archivos con nombres inconsistentes para la misma tarea |

## 3. Diseño técnico

- **`internal/fetch`** con el patrón de los flujos, más pequeño que deploy y QA:
  - `Resolve(cfg, Request{Issue, Space})`: issue obligatorio y en mayúsculas;
    space del flag o del global; vacío busca en todos los spaces, como gtt.
  - `Session.Search()` llena `Pages` (más reciente primero). Sin resultados →
    `ErrNoPages`, envuelto con la clave del issue. Un error de la API se
    devuelve tal cual: no es «sin páginas».
  - `Session.Export(page)` descarga la página (`GetPageWithContent`) y devuelve
    `Export{Page, Text, FileName}` con `BuildIssueTxt`.
  - `Export.Save(path)` escribe el archivo (0644, reemplaza si existe, como
    gtt). Está en el flujo porque lo usan el comando y la TUI.
- **Elegir la página** queda en el llamador. gtt elegía sola la única página
  encontrada y, con varias, pedía un número con 1 por defecto: eso es UI.
- **Nombre de archivo**: `<ISSUE>_<título>.txt`, título reducido a letras,
  dígitos y `_`, máximo 40 caracteres. Ahora translitera tildes, diéresis y
  eñes (tabla mínima, sin dependencias) y, si el título no deja nada, propone
  `<ISSUE>.txt` en vez de `<ISSUE>_.txt`.
- **Diálogos nativos para guardar** (zenity, PowerShell también desde WSL,
  osascript) no se portan aquí: son de la UI. Anotados en la fase 3.

## 4. Archivos

- `internal/fetch/fetch.go` — flujo.
- `internal/fetch/fetch_test.go` — 8 tests con un doble de Atlassian.
- `docs/roadmap.md` — exportar pasa a hecho; diálogos de guardar en fase 3.
- `CLAUDE.md` — `fetch/` en la arquitectura y `ErrNoPages` en los centinelas.

## 5. Cómo verificar

```bash
go test ./internal/fetch -v
bash .claude/skills/cerrar-actividad/check.sh
```

## 6. Pendientes

- Sin probar contra Confluence real: se hará con `draco fetch` (fase 3).
- `internal/git/git.go` sigue diciendo «Verifícalo en Bitbucket» en un mensaje
  al usuario (anotado en 004).
