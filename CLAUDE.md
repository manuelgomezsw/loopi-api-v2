# Git Workflow

## Gitflow — regla obligatoria

Todo cambio en este repositorio debe seguir el flujo **Gitflow**. Nunca hagas cambios directamente en `main` o `develop`.

### Ramas principales

| Rama | Propósito |
|------|-----------|
| `main` | Código en producción. Solo recibe merges desde `release/*` o `hotfix/*`. |
| `develop` | Base de integración. Todo trabajo nuevo parte desde aquí. |

### Crear un branch nuevo

**Siempre parte desde `develop`** (excepto `hotfix/*`, que parte desde `main`):

```bash
git checkout develop
git pull origin develop
git checkout -b <tipo>/<nombre-descriptivo>
```

### Convención de nombres

| Tipo | Prefijo | Cuándo usarlo | Ejemplo |
|------|---------|---------------|---------|
| Nueva funcionalidad | `feature/` | Cualquier nueva feature o mejora | `feature/auth-google-login` |
| Corrección urgente en prod | `hotfix/` | Bug crítico que requiere parche inmediato en `main` | `hotfix/fix-payment-crash` |
| Corrección no urgente | `bugfix/` | Bug detectado en `develop` o QA | `bugfix/fix-empty-cart-error` |
| Preparación de versión | `release/` | Estabilización antes de merge a `main` | `release/v1.2.0` |
| Tareas técnicas / refactor | `chore/` | Dependencias, CI, configuración, refactor | `chore/upgrade-node-20` |

### Reglas

- El nombre del branch debe ser en **minúsculas**, palabras separadas por `-`, en inglés o español consistente con el proyecto.
- Los `hotfix/*` parten desde `main` y se mergean a `main` **y** `develop`.
- Los `feature/*`, `bugfix/*` y `chore/*` parten desde `develop` y se mergean solo a `develop`.
- Los `release/*` parten desde `develop` y se mergean a `main` **y** `develop`.
- Nunca hagas `git push --force` en `main` o `develop`.
