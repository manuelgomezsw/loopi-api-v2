-- Migración 005: Agregar campos de autenticación a empleados
-- Dependencia: migración 003 (tabla empleados) debe estar aplicada
-- El módulo de auth pasa a leer desde `empleados` en lugar de `usuarios`.

SET time_zone = 'America/Bogota';

ALTER TABLE empleados
  ADD COLUMN bloqueado_hasta   DATETIME    NULL    DEFAULT NULL
    COMMENT 'Timestamp hasta el que la cuenta está bloqueada por intentos fallidos'
    AFTER requiere_cambio_contrasena,
  ADD COLUMN intentos_fallidos TINYINT UNSIGNED NOT NULL DEFAULT 0
    COMMENT 'Contador de intentos fallidos de login; resetea a 0 al login exitoso'
    AFTER bloqueado_hasta;
