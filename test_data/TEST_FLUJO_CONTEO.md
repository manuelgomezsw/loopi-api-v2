# 🧪 Testing Flujo Completo: Inventario Conteo

## Preparación

### 1. Ejecutar Script de Test Data

```bash
mysql -u root -p loopi_v2 < test_data/test_inventario_conteo.sql
```

Verifica que se insertaron 10 items TEST-001 a TEST-010.

### 2. Obtener Token JWT

```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@loopi.com",
    "password": "password123"
  }'
```

Guardar el `token` en variable:

```bash
TOKEN="tu_token_jwt_aqui"
TIENDA_ID=1
```

---

## 🔄 FLUJO COMPLETO: Happy Path

### PASO 1: Iniciar Conteo Diario

```bash
curl -X POST http://localhost:8000/api/v1/inventarios \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tienda_id": '$TIENDA_ID',
    "tipo": "diario",
    "horario": "apertura"
  }'
```

**Respuesta esperada (201 Created):**

```json
{
  "id": 123,
  "tienda_id": 1,
  "tipo": "diario",
  "horario": "apertura",
  "estado": "en_progreso",
  "responsable_id": 1,
  "iniciado_en": "2026-07-18T14:30:00Z",
  "items": [
    {
      "id": 1001,
      "item_id": 501,
      "valor_sugerido": 0,
      "valor_esperado": 0,
      "valor_real": null,
      "diferencia": null
    },
    ... (10 items totales)
  ]
}
```

Guardar el `id` del inventario:

```bash
INVENTARIO_ID=123
```

---

### PASO 2: Registrar Valor para Item 1 (TEST-001)

Encontrar el `item_id` del primer item TEST-001 desde la respuesta anterior.

```bash
ITEM_ID=501  # Ajustar con el valor real

curl -X PATCH http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID/items/$ITEM_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "valor_real": 50
  }'
```

**Respuesta esperada (200 OK):**

```json
{
  "id": 1001,
  "item_id": 501,
  "valor_sugerido": 0,
  "valor_esperado": 0,
  "valor_real": 50,
  "diferencia": 50
}
```

**Notas:**
- `diferencia`: valor_real (50) - valor_sugerido (0) = 50
- El frontend debe mostrar "+50" en verde

---

### PASO 3: Registrar Valores para Remaining Items (2-10)

Repetir el PATCH para cada item:

| Item | Código | Valor Real | Diferencia Esperada |
|------|--------|------------|-------------------|
| 1 | TEST-001 | 50 | +50 |
| 2 | TEST-002 | 45 | +45 |
| 3 | TEST-003 | 20 | +20 |
| 4 | TEST-004 | 100 | +100 |
| 5 | TEST-005 | 60 | +60 |
| 6 | TEST-006 | 200 | +200 |
| 7 | TEST-007 | 15 | +15 |
| 8 | TEST-008 | 5 | +5 |
| 9 | TEST-009 | 8 | +8 |
| 10 | TEST-010 | 3 | +3 |

```bash
# Script para registrar todos los items
declare -a ITEM_IDS=(501 502 503 504 505 506 507 508 509 510)  # Ajustar con valores reales
declare -a VALORES=(50 45 20 100 60 200 15 5 8 3)

for i in "${!ITEM_IDS[@]}"; do
  ITEM_ID="${ITEM_IDS[$i]}"
  VALOR="${VALORES[$i]}"
  
  echo "Registrando Item $((i+1)): ID=$ITEM_ID, Valor=$VALOR"
  
  curl -X PATCH http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID/items/$ITEM_ID \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"valor_real\": $VALOR}"
  
  echo ""
done
```

---

### PASO 4: Obtener Detalle del Inventario (Verificar Estado)

Antes de confirmar, verificar que todos los items tengan valor_real:

```bash
curl -X GET http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID \
  -H "Authorization: Bearer $TOKEN"
```

**Validaciones:**
- `estado`: debe ser "en_progreso"
- Todos los items deben tener `valor_real !== null`
- Sumas de diferencias deben ser correctas

---

### PASO 5: Confirmar Conteo

```bash
curl -X POST http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID/confirmar \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Respuesta esperada (200 OK):**

```json
{
  "id": 123,
  "tienda_id": 1,
  "tipo": "diario",
  "horario": "apertura",
  "estado": "completado",
  "responsable_id": 1,
  "iniciado_en": "2026-07-18T14:30:00Z",
  "completado_en": "2026-07-18T14:35:45Z",
  "items": [
    {
      "id": 1001,
      "item_id": 501,
      "valor_sugerido": 0,
      "valor_esperado": 0,
      "valor_real": 50,
      "diferencia": 50
    },
    ... (todos los 10 items)
  ]
}
```

**Validaciones:**
- `estado`: cambió de "en_progreso" a "completado"
- `completado_en`: timestamp actual
- Todos los items incluidos en respuesta

---

## ❌ TESTING DE ERRORES

### Error 1: 409 conteo_duplicado

**Scenario:** Intentar iniciar otro conteo diario/apertura mientras uno está en progreso

```bash
curl -X POST http://localhost:8000/api/v1/inventarios \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tienda_id": '$TIENDA_ID',
    "tipo": "diario",
    "horario": "apertura"
  }'
```

**Respuesta esperada (409 Conflict):**

```json
{
  "error": "conteo_duplicado",
  "mensaje": "Ya existe un conteo en progreso para esta tienda, tipo y horario en esta fecha"
}
```

**Validación Frontend:**
- Mensaje: "Ya existe un conteo en progreso para esta tienda, tipo y horario. Usa la opción Reanudar si deseas continuar."
- Botón "Reanudar" disponible

---

### Error 2: 403 conteo_bloqueado

**Scenario:** Otro usuario intenta registrar valor en conteo que NO inició

```bash
# Obtener token de usuario diferente (user_id = 2)
TOKEN_USER2="token_de_otro_usuario"

curl -X PATCH http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID/items/501 \
  -H "Authorization: Bearer $TOKEN_USER2" \
  -H "Content-Type: application/json" \
  -d '{
    "valor_real": 999
  }'
```

**Respuesta esperada (403 Forbidden):**

```json
{
  "error": "conteo_bloqueado",
  "mensaje": "solo el responsable puede registrar valores"
}
```

**Validación Frontend:**
- Mensaje: "Este conteo está bloqueado. Solo el responsable puede registrar valores."
- Input deshabilitado con candado visual

---

### Error 3: 422 items_sin_registrar

**Scenario:** Intentar confirmar sin registrar todos los items

Resetear los valores a NULL de algunos items (en BD directamente para testing):

```sql
UPDATE inventario_items
SET valor_real = NULL
WHERE inventario_id = $INVENTARIO_ID
AND item_id IN (501, 502, 503);
```

Luego confirmar:

```bash
curl -X POST http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID/confirmar \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Respuesta esperada (422 Unprocessable Entity):**

```json
{
  "error": "items_sin_registrar",
  "mensaje": "todos los items deben tener un valor registrado",
  "detalles": {
    "items_sin_registrar": [501, 502, 503]
  }
}
```

**Validación Frontend:**
- Mensaje: "No todos los items tienen valores registrados. Completa todos antes de confirmar."
- Vuelve a step "register"
- Items sin valor resaltados en rojo

---

### Error 4: 409 ya_completado

**Scenario:** Intentar confirmar dos veces el mismo conteo

Después de confirmar exitosamente (PASO 5), volver a intentar:

```bash
curl -X POST http://localhost:8000/api/v1/inventarios/$INVENTARIO_ID/confirmar \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**Respuesta esperada (409 Conflict):**

```json
{
  "error": "ya_completado",
  "mensaje": "conteo ya está completado"
}
```

**Validación Frontend:**
- Mensaje: "Este conteo ya fue completado anteriormente y no puede ser modificado."

---

### Error 5: 400 horario_required (Validación)

**Scenario:** Intentar crear conteo diario sin especificar horario

```bash
curl -X POST http://localhost:8000/api/v1/inventarios \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tienda_id": '$TIENDA_ID',
    "tipo": "diario"
  }'
```

**Respuesta esperada (400 Bad Request):**

```json
{
  "error": "horario_required",
  "mensaje": "horario es requerido para conteo diario"
}
```

**Validación Frontend:**
- Mensaje: "El horario es obligatorio para conteos diarios."
- Formulario valida antes de enviar

---

## 📊 Validación de Cambio de Estado en UI

### Paso a Paso con DevTools

1. **Paso "select"** → Llenar formulario, click "Iniciar Conteo"
   - Spinner "Iniciando..." durante petición
   - Transición a "register" al recibir 201

2. **Paso "register"** → Registrar valores
   - Spinner "Guardando..." en cada item
   - UI actualiza con valor_real y diferencia (ChangeDetectionStrategy.OnPush ✅)
   - Botón "Continuar" solo habilitado si todos tienen valor

3. **Paso "confirm"** → Resumen
   - Mostrar totales: items contados, tipo, horario
   - Click "Confirmar Conteo"

4. **Paso "complete"** → Éxito
   - Checkmark verde
   - Mensaje: "Conteo Completado"
   - Fecha formateada: "2026-07-18 02:35 p.m"
   - Botón "Nuevo Conteo"

---

## 🔍 Validar Error Mapping

En la consola del navegador, monitorear Network tab:

- **409 Error** → Frontend muestra: *"Ya existe un conteo en progreso..."*
- **403 Error** → Frontend muestra: *"Este conteo está bloqueado..."*
- **422 Error** → Frontend muestra: *"No todos los items tienen valores..."*

No debe haber nunca: *"No se pudo iniciar el conteo. Intenta de nuevo."*

---

## 🧹 Cleanup

Al finalizar testing, ejecutar:

```sql
-- Limpiar datos de prueba
DELETE FROM inventario_items
WHERE inventario_id IN (
  SELECT id FROM inventarios
  WHERE tipo IN ('diario', 'semanal', 'mensual')
  AND DATE(iniciado_en) = CURDATE()
  AND responsable_id = 1
);

DELETE FROM inventarios
WHERE tipo IN ('diario', 'semanal', 'mensual')
AND DATE(iniciado_en) = CURDATE()
AND responsable_id = 1;

DELETE FROM items WHERE codigo LIKE 'TEST-%';
```

---

## ✅ Checklist Final

- [ ] Iniciar conteo: 201 Created ✓
- [ ] Registrar 10 valores: 200 OK x 10 ✓
- [ ] Confirmar conteo: 200 OK, estado=completado ✓
- [ ] 409 conteo_duplicado: Intentar duplicado ✓
- [ ] 403 conteo_bloqueado: Otro usuario ✓
- [ ] 422 items_sin_registrar: Sin completar items ✓
- [ ] 409 ya_completado: Confirmar dos veces ✓
- [ ] UI transitions: select → register → confirm → complete ✓
- [ ] Error messages: Específicos, no genéricos ✓
- [ ] Date format: "yyyy-MM-dd hh:mm a" ✓
- [ ] ChangeDetectionStrategy.OnPush: UI actualiza ✓
- [ ] Cleanup: Datos de test eliminados ✓
