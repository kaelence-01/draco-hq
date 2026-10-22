# 057 — El documento se ve como el de gtt

**Fecha:** 2026-10-22
**Rama:** fix/documento-como-gtt → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor comparó en Confluence un documento de despliegue generado por draco
con uno de referencia hecho a mano, y anotó las diferencias (en archivos
locales que no van al repo). Pidió corregir lo necesario para que el
documento se vea como el de referencia, que es como lo publicaba gtt.

## 2. Motivación

| Diferencia | ¿gtt lo hacía? | Daño real |
|---|---|---|
| La primera columna de Épica / Tarea(s) ocupa media página | Sí (`colwidth` 63/697, layout `default`, `width` 1800) | El enlace a la tarea queda desplazado; la tabla de archivos reparte cinco columnas iguales |
| Archivos y enlaces «link» sin viñeta | Sí (`bulletList` en «Nombre del archivo» y «Observación») | Las filas con varios archivos se leen como un bloque |
| «A considerar» suelto, fuera del recuadro | Sí (tabla de una celda con la lista de tareas) | El checklist no se distingue del resto de la página |
| Mención @ del responsable en cada paso | No | — (función nueva, fuera de esta actividad) |
| Pasos distintos a la referencia | Salen de `deploy_checklist` | — (es configuración, no un error) |

Las tres primeras eran **pérdidas aceptadas** al pasar el documento a
Markdown en la fase 2 (`docs/arquitectura/proveedores.md` §5, actividad 4b,
commit `3aac582`). Los anchos se habían dejado «por plantilla en la
actividad 5», que no los incluyó. La comparación del autor muestra que son
justo lo que hace que el documento se vea como el de gtt.

## 3. Diseño técnico

- **Las plantillas siguen siendo Markdown.** Lo que Markdown no expresa se
  recupera con convenciones que entiende `atlassian.MarkdownToADF`, como las
  que ya había (`<br>`, celda en negrita, `<url>`):
  - **`<!-- draco:tabla anchos=63,697 [ancho=1800] [layout=default] -->`**
    antes de una tabla: `colwidth` en cada celda, y `layout` y `width` en la
    tabla. Los valores por defecto son los de gtt.
  - **`<!-- draco:recuadro -->`** antes de un bloque: lo envuelve en una
    tabla de una celda, como `considerTable` de gtt.
  - **Una celda cuyas líneas empiezan todas con `- `** (separadas por
    `<br>`) es una lista con viñetas.
- **Por qué comentarios HTML:** en un destino Markdown no se ven, la
  plantilla sigue siendo legible, y el texto del documento no cambia.
  Alternativas descartadas:
  - anchos en `template.yaml`: separan el ancho de la tabla a la que
    pertenece, y no sirven para el recuadro;
  - inferir los anchos en el conversor (por ejemplo, «primera columna en
    negrita es angosta»): adivinar en el adaptador es justo lo que la
    arquitectura evita.
- **Un comentario `draco:` mal escrito** (nombre desconocido, ancho que no
  es número) no es directiva: queda como texto literal, igual que cualquier
  otro HTML, para que se vea en la página y se corrija.
- **Plantillas incluidas:**
  - despliegue: anchos de gtt en las dos tablas (63/697 y
    69/175/176/188/152), viñetas en archivos y enlaces (también el «—» sin
    VCS, como gtt), y el checklist en un recuadro;
  - consolidado QA: los anchos que tenía gtt en «Información General»
    (200/560, `full-width`, 760) y en la tabla consolidada
    (150/120/120/120/120/130/130, `full-width`, 800). «Resumen» cambió de
    estructura en draco (la observación es una sección), así que queda sin
    anchos.
- **Documentos ya publicados:** al actualizarlos, «A considerar» se
  conserva tal como está en la página (sin recuadro si se creó antes). El
  resto del cuerpo se regenera con el formato nuevo.
- **Exportar** procesa las tareas dentro de una celda igual que en las
  páginas que publicaba gtt, que ya venían así.
- **Goldens:** cambian los Markdown del documento y del consolidado (los
  comentarios y las viñetas) y los ADF publicados (`adf-deploy`, `adf-qa`).
  El diff se revisó: solo agrega `colwidth`, `layout`/`width`, `bulletList`
  y la tabla del recuadro.

## 4. Archivos

- `internal/provider/atlassian/markdown.go`, `markdown_test.go` —
  directivas, viñetas en celdas, tests y semillas del fuzz.
- `internal/document/templates/deploy/main.md.tmpl`,
  `templates/qa/main.md.tmpl` — las convenciones en las plantillas
  incluidas.
- Goldens: `internal/document/testdata/*.golden.md`,
  `internal/deploy/testdata/draft-create.golden.md`,
  `internal/qa/testdata/draft-create-sprint.golden.md`,
  `internal/provider/atlassian/testdata/adf-*.golden.json`.
- `docs/arquitectura/proveedores.md` §5 — convenciones y pérdidas.
- `docs/guia-usuario.md` — las convenciones para quien escribe plantillas.
- `docs/bitacora/fase-6/057-documento-como-gtt.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/provider/atlassian run='BulletCell|DracoDirectives|GoldenPublishedADF'
draco template preview deploy        # el Markdown con las convenciones
```

A mano (pendiente del autor): `draco g … -p <proyecto>` contra una página de
prueba en Confluence, y compararla con la de referencia: columna angosta en
Épica / Tarea(s), viñetas en archivos y enlaces, checklist en un recuadro.

## 6. Pendientes

- **Mención del responsable** en cada paso de «A considerar»: necesita el
  `accountId` (el asignado de la tarea en Jira es lo natural, y el puerto
  `Tracker` todavía no lo devuelve). Decisión del autor.
- El gris del texto guía de Épica sigue sin recuperarse; la comparación lo
  da por bueno.
