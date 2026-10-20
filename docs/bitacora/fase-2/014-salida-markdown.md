# 014 — Destino Markdown local

**Fecha:** 2026-10-19
**Rama:** feature/salida-markdown → develop
**Fase del roadmap:** 2

## 1. Solicitud

Seguir con el siguiente paso de la fase 2. Se tomó la actividad 6 de la
secuencia de proveedores (destino Markdown local) antes que la 5 (plantillas),
como se recomendó: es chica y es la primera prueba real de que el diseño de
puertos aguanta un segundo destino.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Confluence era el único destino | Un puerto con forma de Confluence pasa inadvertido hasta que llega otro destino; descubrirlo después de las plantillas sale más caro |
| Equipos sin Confluence | No pueden usar draco, aunque solo necesiten el documento junto al código |

## 3. Diseño técnico

- **`markdown.Store`** implementa `provider.DocStore` sobre una carpeta:
  - documento = archivo `.md`, primera línea `# <título>`, ID = ruta
    relativa con `/`; el nombre del archivo sale del título (minúsculas,
    sin tildes, guiones, hasta 80 caracteres) y no pisa uno existente
    (`-2`, `-3`…);
  - `FindByTitle` exacto; `FindByTitleKey` exige la clave como palabra
    (`OPS-7` no encuentra `OPS-70`); `Search` en el contenido y
    `SearchTitle` en el título, el más reciente primero;
  - `Locations` devuelve **carpetas**: las que tienen documentos con el
    prefijo, más la raíz. A diferencia de Confluence, una carpeta vacía
    recibe el primer documento sin crear nada a mano. Los archivos no tienen
    autor: `AuthorID` se ignora;
  - `Sections`/`Update` cortan el Markdown por títulos (el mismo criterio:
    hasta el siguiente título del mismo nivel o superior), sin contar los que
    están dentro de bloques de código. Una sección vacía existe;
  - `Update` no escribe si no puede leer el archivo actual;
  - `Export` devuelve el cuerpo sin la línea del título, con
    `Source: "Markdown local"`.
- **Seguridad:** toda ruta se resuelve dentro de la carpeta (un ID con `..`
  se rechaza) y cada escritura va a un temporal que se renombra al final.
- **`provider.NormalizeHeading`:** la comparación de títulos de sección pasa
  a `provider` para que Confluence y Markdown reconozcan las secciones igual.
- **Hallazgo sobre los puertos:** no hizo falta cambiar ninguno. Lo único
  propio de Confluence era que `Locations` devolviera páginas hermanas; con
  `DocRef` de carpetas (ID terminado en `/`) el flujo no nota la diferencia.
- **Criterio de cierre:** `TestDeployDocumentWithoutConfluence` corre el
  flujo `deploy` real contra la carpeta: publica, se edita «A considerar» a
  mano en el archivo y la segunda publicación actualiza el mismo archivo
  conservando la edición.
- **Batería común de contrato:** no se escribió. Correrla sobre Confluence
  exige un Confluence simulado completo; cada adaptador tiene sus pruebas y el
  flujo corre de punta a punta sobre Markdown. Se decide con un tercer
  destino.

## 4. Archivos

- `internal/provider/markdown/markdown.go` — adaptador (nuevo).
- `internal/provider/markdown/markdown_test.go`, `deploy_test.go` — pruebas
  del adaptador y del flujo de punta a punta.
- `internal/provider/provider.go` — `NormalizeHeading`.
- `internal/provider/atlassian/preserve.go` — usa `provider.NormalizeHeading`.
- `docs/arquitectura/proveedores.md`, `docs/roadmap.md`, `CLAUDE.md`.

## 5. Cómo verificar

```bash
make test pkg=./internal/provider/markdown v=1
make test
```

## 6. Pendientes

- Elegir el destino por config (`docs: { provider: markdown, path: … }`):
  actividad 7, perfiles.
- Con Jira como tracker y Markdown como destino, el enlace remoto apunta a una
  ruta local: verificar en la fase 3 qué acepta Jira o usar la URL del
  archivo en el VCS.
