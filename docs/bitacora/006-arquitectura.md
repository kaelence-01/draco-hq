# 006 — Diseño de proveedores y plantillas

**Fecha:** 2026-10-19
**Rama:** docs/arquitectura → develop
**Fase del roadmap:** 2

## 1. Solicitud

Que draco no quede atado solo a Atlassian: que funcione con otras herramientas
además de Jira (y Confluence), en particular GitHub, GitLab, Azure DevOps,
Linear y Notion. Hacerlo ahora, antes del modo comando. La base es una
propuesta previa del autor para gtt (expansión multiplataforma y plantillas
configurables), documento privado.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los flujos dependen de tipos de `atlassian` y el documento se arma en ADF | Un equipo sin Jira ni Confluence no puede usar draco |
| La estructura del documento está en código | Cambiar un título o una sección exige recompilar y liberar |
| El comando y la TUI todavía no existen | Si se construyen sobre tipos de Atlassian, el desacople después obliga a reescribir las dos interfaces |

## 3. Diseño técnico

El diseño completo queda en `docs/arquitectura/proveedores.md`. Decisiones
principales:

- **Roles separados y combinables:** tracker, destino de documentos, fuente de
  cambios, plantillas y notificación.
- **Puertos y adaptadores:** modelo neutro e interfaces en `internal/provider`;
  Jira/Confluence pasan a `internal/provider/atlassian`. Un flujo nunca importa
  un adaptador. Adaptadores compilados en el binario; capacidades opcionales
  (QA de Jira) como interfaces aparte.
- **Plantilla → Markdown → formato del destino**, con `text/template` y
  `goldmark` (dependencia aprobada por el autor frente a escribir un parser
  propio).
- **Ajustes frente a la propuesta original:**
  - la preservación de secciones escritas a mano es parte del puerto y la hace
    el adaptador en su formato nativo (la propuesta no la contemplaba);
  - la unidad sigue siendo el issue; un documento por release queda como
    extensión;
  - QA queda solo para Jira al inicio;
  - el criterio «documento idéntico» solo aplica al pasar a puertos: Markdown
    no expresa anchos de tabla, tarjetas de enlace, emojis ni texto gris, así
    que desde el paso a Markdown el criterio es «contenido equivalente» con
    esas diferencias listadas y aceptadas;
  - ejemplos genéricos: nada de los datos de la propuesta privada.
- **Roadmap:** los pendientes de la fase 2 se reemplazan por 7 actividades
  (esta es la 1). «Convenciones de texto configurables» queda dentro de
  plantillas, «proyectos y configuración» dentro de perfiles, y «listar
  commits» dentro del puerto de cambios. Retrasa el modo comando a cambio de
  construir el comando y la TUI una sola vez sobre la base definitiva.

## 4. Archivos

- `docs/arquitectura/proveedores.md` — diseño (nuevo).
- `docs/roadmap.md` — visión multiplataforma, fase 2 reordenada en 7
  actividades con criterio de cierre, tres decisiones nuevas, pendientes de
  proveedores.
- `CLAUDE.md` — descripción, arquitectura de `docs/`, `goldmark` en
  dependencias aprobadas, nota de transición a puertos.

## 5. Cómo verificar

```bash
bash .claude/skills/cerrar-actividad/check.sh
```

Revisión: leer `docs/arquitectura/proveedores.md` antes de aprobar el PR; las
interfaces son borrador y se fijan en la actividad 3.

## 6. Pendientes

- Mapeo de estados entre trackers, gestor de secretos del sistema y plantillas
  remotas (decisiones pendientes del roadmap).
