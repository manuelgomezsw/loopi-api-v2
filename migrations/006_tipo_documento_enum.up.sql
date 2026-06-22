-- Migración 006: Cambiar tipo_documento de VARCHAR(30) a ENUM('CC','CE','NUIP','PE')
-- Dependencia: migración 003 (tabla empleados) debe estar aplicada.
--
-- Paso 1: Nullificar valores que no pertenecen al nuevo ENUM (datos legacy de texto libre).
-- Paso 2: Alterar la columna a ENUM.

SET time_zone = 'America/Bogota';

UPDATE empleados
SET tipo_documento = NULL
WHERE tipo_documento IS NOT NULL
  AND tipo_documento NOT IN ('CC', 'CE', 'NUIP', 'PE');

ALTER TABLE empleados
  MODIFY COLUMN tipo_documento ENUM('CC', 'CE', 'NUIP', 'PE') NULL;
