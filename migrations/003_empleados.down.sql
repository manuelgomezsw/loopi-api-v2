-- Rollback 003: Eliminar tabla empleados
-- Ejecutar DESPUÉS de 004_log_auditoria_empleados.down.sql
DROP TABLE IF EXISTS empleados;
