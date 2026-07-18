-- ========================================================
-- TEST SCRIPT: Inventario Conteo - Validación Flujo Completo
-- ========================================================
-- Propósito: Insertar 10 items de prueba para validar:
--   1. Iniciar conteo
--   2. Registrar valores
--   3. Confirmar conteo
--   4. Manejo de errores (409, 403, 422, etc)
-- ========================================================

-- Limpiar datos previos de prueba (OPCIONAL - comentar si no deseas)
-- DELETE FROM inventario_items WHERE inventario_id IN (SELECT id FROM inventarios WHERE tipo IN ('test_diario', 'test_semanal'));
-- DELETE FROM inventarios WHERE tipo IN ('test_diario', 'test_semanal');
-- DELETE FROM items WHERE codigo LIKE 'TEST-%';

-- ========================================================
-- PASO 1: Insertar 10 items de prueba
-- ========================================================

INSERT INTO items (codigo, nombre, tipo, subcategoria_id, proveedor_id, unidad_medida_id, costo_unitario, frecuencia_inventario, stock_seguridad, activo, creado_por, creado_en, actualizado_por, actualizado_en)
VALUES
  ('TEST-001', '[TEST] Vaso 9oz - Prueba 1', 'insumo', 1, NULL, 3, 150.00, 'diario', 10, 1, 1, NOW(), 1, NOW()),
  ('TEST-002', '[TEST] Vaso 12oz - Prueba 2', 'insumo', 1, NULL, 3, 200.00, 'diario', 15, 1, 1, NOW(), 1, NOW()),
  ('TEST-003', '[TEST] Café Paz x250g - Prueba 3', 'material_consumo', 6, 1, 1, 28000.00, 'diario', 5, 1, 1, NOW(), 1, NOW()),
  ('TEST-004', '[TEST] Servilletas - Prueba 4', 'insumo', 1, 2, 3, 50.00, 'semanal', 20, 1, 1, NOW(), 1, NOW()),
  ('TEST-005', '[TEST] Pitillos x200 - Prueba 5', 'insumo', 1, 21, 3, 100.00, 'semanal', 5, 1, 1, NOW(), 1, NOW()),
  ('TEST-006', '[TEST] Bolsas papel #1 - Prueba 6', 'insumo', 1, NULL, 3, 75.00, 'semanal', 30, 1, 1, NOW(), 1, NOW()),
  ('TEST-007', '[TEST] Leche Entera (bolsa) - Prueba 7', 'material_consumo', 13, 8, 3, 3500.00, 'diario', 10, 1, 1, NOW(), 1, NOW()),
  ('TEST-008', '[TEST] Chocolate en Polvo - Prueba 8', 'material_consumo', 8, 9, 3, 15000.00, 'mensual', 2, 1, 1, NOW(), 1, NOW()),
  ('TEST-009', '[TEST] Helado x 5 litros - Prueba 9', 'material_consumo', 12, 10, 3, 25000.00, 'semanal', 3, 1, 1, NOW(), 1, NOW()),
  ('TEST-010', '[TEST] Miel 250g - Prueba 10', 'material_consumo', 10, 14, 1, 12000.00, 'mensual', 1, 1, 1, NOW(), 1, NOW());

-- ========================================================
-- PASO 2: Crear inventario de prueba (ESTADO: en_progreso)
-- ========================================================
-- Este paso se ejecutará desde la API, NO manualmente.
-- Pero dejamos la estructura para referencia:
--
-- INSERT INTO inventarios (tienda_id, fecha, tipo, horario, estado, responsable_id, iniciado_en, creado_en, actualizado_en)
-- VALUES (1, CURDATE(), 'diario', 'apertura', 'en_progreso', 1, NOW(), NOW(), NOW());
--
-- OBTENER EL inventario_id y luego asociar los 10 items:
--
-- INSERT INTO inventario_items (inventario_id, item_id, valor_sugerido, valor_esperado, valor_real, diferencia, creado_en, actualizado_en)
-- SELECT <inventario_id>, id, 0, 0, NULL, NULL, NOW(), NOW()
-- FROM items WHERE codigo LIKE 'TEST-%';

-- ========================================================
-- PASO 3: Verificar items insertados
-- ========================================================

SELECT
  id,
  codigo,
  nombre,
  tipo,
  frecuencia_inventario,
  stock_seguridad,
  activo,
  creado_en
FROM items
WHERE codigo LIKE 'TEST-%'
ORDER BY id DESC
LIMIT 10;

-- ========================================================
-- GUÍA DE TESTING DESDE FRONTEND
-- ========================================================
--
-- 1. INICIAR CONTEO (POST /api/v1/inventarios)
--    Body:
--    {
--      "tienda_id": 1,
--      "tipo": "diario",
--      "horario": "apertura"
--    }
--    Esperado: 201 Created con 10 items TEST-%
--
-- 2. REGISTRAR VALORES (PATCH /api/v1/inventarios/{id}/items/{item_id})
--    Registrar valor_real para cada item:
--    - TEST-001: 50 (valor_sugerido: 0, diferencia: 50)
--    - TEST-002: 45 (valor_sugerido: 0, diferencia: 45)
--    - TEST-003: 20 (valor_sugerido: 0, diferencia: 20)
--    - TEST-004: 100 (valor_sugerido: 0, diferencia: 100)
--    - TEST-005: 60 (valor_sugerido: 0, diferencia: 60)
--    - TEST-006: 200 (valor_sugerido: 0, diferencia: 200)
--    - TEST-007: 15 (valor_sugerido: 0, diferencia: 15)
--    - TEST-008: 5 (valor_sugerido: 0, diferencia: 5)
--    - TEST-009: 8 (valor_sugerido: 0, diferencia: 8)
--    - TEST-010: 3 (valor_sugerido: 0, diferencia: 3)
--
-- 3. CONFIRMAR CONTEO (POST /api/v1/inventarios/{id}/confirmar)
--    Esperado: 200 OK con estado: "completado"
--
-- 4. VALIDAR ERRORES:
--    a) 409 conteo_duplicado:
--       - Intentar iniciar otro conteo diario/apertura mientras esté en progreso
--    b) 403 conteo_bloqueado:
--       - Con otro usuario, intentar registrar valor en conteo activo
--    c) 422 items_sin_registrar:
--       - Intentar confirmar sin registrar todos los items
--    d) 409 ya_completado:
--       - Intentar confirmar dos veces el mismo conteo
--
-- ========================================================
-- CONSULTA PARA LISTAR ITEMS DE PRUEBA
-- ========================================================

-- Contar items de prueba
SELECT COUNT(*) as total_items_prueba
FROM items
WHERE codigo LIKE 'TEST-%';

-- Listar con detalles
SELECT
  id,
  codigo,
  nombre,
  tipo,
  subcategoria_id,
  unidad_medida_id,
  frecuencia_inventario,
  stock_seguridad,
  activo
FROM items
WHERE codigo LIKE 'TEST-%'
ORDER BY id;

-- ========================================================
-- CLEANUP (EJECUTAR AL FINAL DEL TESTING)
-- ========================================================
-- Descomentar para limpiar datos de prueba:
--
-- DELETE FROM inventario_items
-- WHERE inventario_id IN (
--   SELECT id FROM inventarios
--   WHERE tipo IN ('test_diario', 'test_semanal')
-- );
--
-- DELETE FROM inventarios
-- WHERE tipo IN ('test_diario', 'test_semanal');
--
-- DELETE FROM items WHERE codigo LIKE 'TEST-%';
--
-- ========================================================
