# 027 — Cierre de la fase 3

**Fecha:** 2026-10-19
**Rama:** docs/cierre-fase-3 → develop
**Fase del roadmap:** 3

## 1. Solicitud

Cerrar la fase 3 después del primer release: registrar la verificación de
`v0.1.0` y preparar la prueba de actualización.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El pie de `draco help` decía «Estado y próximos comandos: docs/roadmap.md (fase 3)» | Quien instala el binario no tiene el repo: la referencia no lleva a ningún lado y quedaba vieja en cada fase |
| La prueba de actualización necesita una versión nueva | Un `v0.1.1` sin cambios no muestra que `draco update` instaló otro binario |

## 3. Diseño técnico

- **Verificado con `v0.1.0`:**
  - el release no es pre-release y trae los 5 binarios y `checksums.txt`;
  - la descarga desde `releases/latest/download/…` (los enlaces del README)
    verifica con `sha256sum -c` y el binario lleva `v0.1.0`;
  - el autor ejecutó el binario descargado: se instaló, quedó en el PATH y
    `draco version` / `draco help` responden desde una terminal nueva;
  - con el código en `dev`, `draco update --check` encuentra `v0.1.0` y no
    reemplaza nada.
- **Pie de la ayuda**: ahora apunta al repositorio
  (`https://github.com/` + `updater.Repo`), que vale para cualquier versión.
  El test de la ayuda lo exige y rechaza la ruta del repo.
- **Prueba de actualización**: esta actividad se publica como `v0.1.1`
  (`make release bump=patch`). `draco update` desde `v0.1.0` debe instalarla,
  y el pie nuevo de `draco help` muestra que el binario cambió.
- Fase 3 marcada como cerrada en el roadmap. Las pruebas manuales de `g`,
  `qa` y `f` contra una instancia real siguen a cargo del autor (bitácoras
  019 y 020); un error ahí se corrige con un `fix/` y un patch.

## 4. Archivos

- `cmd/help.go`, `cmd/app_test.go` — pie de la ayuda general.
- `docs/roadmap.md` — fase 3 cerrada, con su cierre.
- `docs/bitacora/fase-3/027-cierre-fase-3.md`, índice.

## 5. Cómo verificar

```bash
make run args="help" | tail -2     # Documentación y versiones: https://github.com/kaelence-01/draco-hq
# después del merge a main:
make release bump=patch            # v0.1.1
draco update --check && draco update && draco version   # en la terminal, con v0.1.0 instalado
```

## 6. Pendientes

- Pruebas manuales de `g`, `qa` y `f` contra la instancia del autor.
- Fase 4: modo visual (plan antes de escribir código).
