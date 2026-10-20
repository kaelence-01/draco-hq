# 017 — README y plan de la fase 3

**Fecha:** 2026-10-19
**Rama:** docs/readme → develop
**Fase del roadmap:** 3

## 1. Solicitud

Arrancar la fase 3 y agregar un README: el repositorio no tenía.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Sin README | Quien llega al repo público no sabe qué es draco, en qué estado está ni cómo se trabaja en él |
| La fase 3 tenía una lista de tareas, sin actividades ni criterios | No había forma de saber qué cierra la fase, como sí la hubo en la fase 2 |

## 3. Diseño técnico

- **README** en español, como la interfaz: qué hace (las cuatro funciones),
  las dos formas de uso, el estado actual **sin prometer lo que no existe**
  (todavía no hay binario; los ejemplos de comandos son la interfaz prevista),
  perfiles y plantillas con un ejemplo genérico, desarrollo con el Makefile,
  estructura, cómo se trabaja (ramas, un commit, bitácora, sin datos de una
  organización), origen en gtt y licencia MIT. La demo interactiva es privada
  y no se enlaza.
- **Roadmap:** la fase 3 pasa a «en curso» con 7 actividades y su criterio de
  cierre: README, base del comando, `draco g`, `qa` y `f`, `backlog`, `init`
  y configuración, y release (instalador, updater, checksums). El README
  completo con la instalación sigue en v1.0.
- Primera entrada de la fase 3: crea `docs/bitacora/fase-3/` y su sección en
  el índice.

## 4. Archivos

- `README.md` — nuevo.
- `docs/roadmap.md` — fase 3 en curso y su plan; v1.0 completa el README.
- `docs/bitacora/fase-3/017-readme.md`, `docs/bitacora/README.md`.

## 5. Cómo verificar

```bash
make check
```

Leer el README renderizado en GitHub después del merge.
