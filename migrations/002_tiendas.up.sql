-- Migración 002: Crear tabla tiendas
-- Dependencia: migración 001 (tabla usuarios) debe estar aplicada

SET time_zone = 'America/Bogota';

CREATE TABLE IF NOT EXISTS tiendas (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  codigo          VARCHAR(20)     NOT NULL,
  nombre          VARCHAR(150)    NOT NULL,
  direccion       VARCHAR(255)    NOT NULL,
  ciudad          VARCHAR(100)    NOT NULL,
  telefono        VARCHAR(20)     NOT NULL,
  activo          TINYINT(1)      NOT NULL DEFAULT 1,
  creado_por      BIGINT UNSIGNED NOT NULL,
  creado_en       DATETIME        NOT NULL,
  actualizado_por BIGINT UNSIGNED NOT NULL,
  actualizado_en  DATETIME        NOT NULL,

  PRIMARY KEY (id),
  UNIQUE KEY uq_tiendas_codigo  (codigo),
  UNIQUE KEY uq_tiendas_nombre  (nombre),
  INDEX       idx_tiendas_activo (activo),

  CONSTRAINT fk_tiendas_creado_por
    FOREIGN KEY (creado_por)      REFERENCES usuarios (id),
  CONSTRAINT fk_tiendas_actualizado_por
    FOREIGN KEY (actualizado_por) REFERENCES usuarios (id)

) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;
