# Documento de Despliegue - OPS-7 - Filtro de pagos- rango (fechas)

<!-- draco:tabla anchos=63,697 -->
|  |  |
|---|---|
| **Épica** | _Enlace a épica o función de Jira relacionada_ |
| **Tarea(s)** | <https://example.test/browse/OPS-7> |

## Arquitecturas e interfaces

### Proyectos y formularios - Frontend:

<!-- draco:tabla anchos=69,175,176,188,152 -->
| Servidor | Aplicación web | Ubicación | Nombre del archivo | Observación |
|---|---|---|---|---|
| [pagos-web](https://bitbucket.org/mi-org/pagos-web) | pagos-web | src/pagos | - filtro.component.ts | - [link](https://bitbucket.org/mi-org/pagos-web/commits/e4f5a6b#chg-src/pagos/filtro.component.ts) |

### Proyectos y formularios - Backend:

<!-- draco:tabla anchos=69,175,176,188,152 -->
| Servidor | Aplicación web | Ubicación | Nombre del archivo | Observación |
|---|---|---|---|---|
| [pagos-api](https://bitbucket.org/mi-org/pagos-api) | pagos-api | . | - composer.json | - [link](https://bitbucket.org/mi-org/pagos-api/commits/a1b2c3d#chg-composer.json) |
| [pagos-api](https://bitbucket.org/mi-org/pagos-api) | pagos-api | app/Http/Pagos | - FiltroController.php<br>- Rango.php | - [link](https://bitbucket.org/mi-org/pagos-api/commits/a1b2c3d#chg-app/Http/Pagos/FiltroController.php)<br>- [link](https://bitbucket.org/mi-org/pagos-api/commits/a1b2c3d#chg-app/Http/Pagos/Rango.php) |

## A considerar:

<!-- draco:recuadro -->
- [ ] Respaldar base de datos
- [ ] Desplegar backend
