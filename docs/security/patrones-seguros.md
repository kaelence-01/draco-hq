# Patrones seguros — draco

> **Documento vivo.** Cada vez que una revisión encuentre un anti-patrón, se agrega aquí con su motivo, el patrón correcto y la regla.
>
> Hereda los patrones P-001 a P-008 de gtt, el proyecto del que nace draco. Los ejemplos están reescritos con valores genéricos: este repositorio es público.

| ID | Tema |
|---|---|
| [P-001](#p-001--urls-de-instancia-fuera-del-código) | URLs de instancia fuera del código |
| [P-002](#p-002--las-variables-de-entorno-configuran-no-saltan-controles) | Las variables de entorno configuran, no saltan controles |
| [P-003](#p-003--identificadores-de-atlassian-config--override) | Identificadores de Atlassian: config + override |
| [P-004](#p-004--host-y-organización-del-vcs-por-proyecto) | Host y organización del VCS por proyecto |
| [P-005](#p-005--ruta-absoluta-antes-de-ejecutar-un-binario) | Ruta absoluta antes de ejecutar un binario |
| [P-006](#p-006--verificar-la-integridad-de-lo-descargado) | Verificar la integridad de lo descargado |
| [P-007](#p-007--secretos-nunca-palabras-como-token-sí) | Secretos nunca; palabras como «token», sí |
| [P-008](#p-008--datos-de-personas-y-de-la-organización-fuera-del-binario) | Datos de personas y de la organización fuera del binario |

Además, dos reglas transversales:

- **Todo valor interpolado en JQL o CQL pasa por `atlassian.quoteLiteral`.** Nunca se escribe un valor entre comillas a mano en un format string: un valor con comillas rompe la consulta o, peor, cambia lo que pregunta.
- **Una consulta fallida nunca se publica como un control fallido.** El reporte de QA es evidencia: si no se pudo verificar algo, se muestra como desconocido (`QAIssue.DeployDocUnknown`), no como ✗.

---

## P-001 — URLs de instancia fuera del código

Una URL de instancia (`https://<empresa>.atlassian.net`, una intranet) escrita en el código ata el binario a una organización, entrega su topología a quien lea el repo o haga `strings` sobre el ejecutable, y algunos EDR la marcan como indicador de un agente C2.

**Regla:** ninguna URL de una instancia, SaaS privado o intranet aparece como literal en el código. La ingresa el usuario (`draco init`) y vive en `config.yaml`. En textos de ayuda se usa un ejemplo genérico (`https://empresa.atlassian.net`).

## P-002 — Las variables de entorno configuran, no saltan controles

**Regla:** las variables de entorno sirven para configuración (credenciales, URL base, space). Nunca para evadir autenticación, autorización o verificación de integridad, ni siquiera "solo en desarrollo".

## P-003 — Identificadores de Atlassian: config + override

Space key, project key, IDs de estado y de campos personalizados pertenecen a la instancia del usuario.

**Regla:** nunca son literales en el código. Viven en tres capas cuando aplica: valor global en `Config`, override por proyecto en `ProjectConfig` y flag para un comando puntual. El reporte de QA lee los suyos de `qa_jira` y los valida antes de consultar (`QAJiraConfig.Validate`).

```yaml
qa_jira:
  project: APP
  statuses: [10001, 10002, 10003, 10004]   # ids o nombres
  testing_status: Testing                  # opcional; por defecto «Testing»
  sprint_name: "{module}_Sprint {sprint}"  # opcional; así se llama el sprint en Jira
  fields:
    coding_errors: customfield_10001
    dev_returns: customfield_10002
    dev_status: customfield_10003
```

## P-004 — Host y organización del VCS por proyecto

El host y el workspace del VCS escritos en el código revelan la estructura de repositorios de una organización, y si cambian rompen todos los enlaces.

**Regla:** `vcs_host` y `vcs_org` pertenecen a `ProjectConfig`, con override puntual por flag. `internal/document` no conoce ningún VCS concreto: recibe los datos ya resueltos. Si un nombre de repo no está configurado se deduce del directorio del clon, nunca de un literal.

## P-005 — Ruta absoluta antes de ejecutar un binario

**Regla:** toda llamada a `exec.Command` con un ejecutable del sistema usa antes `exec.LookPath` para obtener su ruta absoluta, y si no está instalado lo dice con un mensaje claro. Evita ejecutar un binario homónimo plantado en el directorio actual o en el PATH.

## P-006 — Verificar la integridad de lo descargado

**Regla:** toda descarga de un ejecutable se verifica con SHA-256 contra el `checksums.txt` publicado en el mismo release, antes de escribirlo en disco. El proceso de release genera y publica ese archivo.

## P-007 — Secretos nunca; palabras como «token», sí

**Regla:** ningún secreto real (token, contraseña, API key) aparece en el código ni en constantes. Los textos de ayuda que mencionan las palabras «token» o «password» son normales: si un scanner los reporta, se documenta el falso positivo con el string exacto. Un ejemplo genérico de configuración, como un nombre de workspace, sigue siendo topología (P-001), no un falso positivo.

## P-008 — Datos de personas y de la organización fuera del binario

Los binarios se publican: lo que se compila dentro (nombres del equipo, pasos de despliegue de un stack, convenciones de una empresa) queda a la vista de cualquiera, y cambiarlo obliga a publicar una versión nueva.

**Regla:** ningún nombre de persona, cargo, cliente, proveedor o identificador de la organización es un literal en el código. Si un documento lo necesita, se inyecta desde `config.yaml` y, si falta, se muestra como `—`; nunca un valor por defecto que parezca real. Antes de escribir un literal en una plantilla, pregúntate si cambiaría al cambiar de persona, de equipo o de cliente: si la respuesta es sí, es configuración.

---

## Cómo agregar un patrón

1. Asignar el siguiente ID (`P-009`, …) y agregarlo a la tabla de arriba.
2. Explicar por qué es un problema, el patrón correcto y la **Regla** en una oración accionable.
3. Si agrega un valor configurable, indicar en qué capa vive (global, proyecto, flag).
4. Usar siempre ejemplos genéricos: nada de nombres, repos ni URLs reales.
