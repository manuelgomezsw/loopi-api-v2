-- Rollback 005: Eliminar campos de autenticación de empleados
ALTER TABLE empleados
  DROP COLUMN intentos_fallidos,
  DROP COLUMN bloqueado_hasta;
