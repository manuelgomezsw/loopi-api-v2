-- Migración 006 (revert): Revertir tipo_documento de ENUM a VARCHAR(30)
-- Nota: los valores que fueron nullificados en el .up no se restauran.

SET time_zone = 'America/Bogota';

ALTER TABLE empleados
  MODIFY COLUMN tipo_documento VARCHAR(30) NULL;
