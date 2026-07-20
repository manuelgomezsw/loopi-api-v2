-- Rollback 004: Eliminar tabla log_auditoria_empleados
-- Ejecutar ANTES de 003_empleados.down.sql (tiene FK a empleados)
DROP TABLE IF EXISTS log_auditoria_empleados;
