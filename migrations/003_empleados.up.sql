-- Migración 003: Crear tabla empleados
-- Dependencia: migración 002 (tabla tiendas) debe estar aplicada

SET time_zone = 'America/Bogota';

CREATE TABLE IF NOT EXISTS empleados (
  id                         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  nombre                     VARCHAR(100)    NOT NULL,
  apellido                   VARCHAR(100)    NOT NULL,
  usuario                    VARCHAR(50)     NOT NULL
    COMMENT 'Único en el sistema; inmutable tras creación',
  contrasena_hash            VARCHAR(72)     NOT NULL
    COMMENT 'bcrypt hash (cost 12); max 72 bytes útiles para bcrypt',
  rol                        ENUM('admin', 'lider_tienda', 'barista') NOT NULL,
  tienda_id                  BIGINT UNSIGNED NULL
    COMMENT 'NULL solo para rol=admin',
  tipo_documento             VARCHAR(30)     NULL,
  numero_documento           VARCHAR(30)     NULL,
  telefono                   VARCHAR(20)     NULL,
  email                      VARCHAR(150)    NULL,
  fecha_nacimiento           DATE            NULL,
  activo                     TINYINT(1)      NOT NULL DEFAULT 1,
  requiere_cambio_contrasena TINYINT(1)      NOT NULL DEFAULT 1
    COMMENT '1 = debe cambiar en próximo login',
  creado_en                  DATETIME        NOT NULL,
  actualizado_en             DATETIME        NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_empleados_usuario UNIQUE (usuario),
  CONSTRAINT fk_empleados_tienda  FOREIGN KEY (tienda_id) REFERENCES tiendas(id),

  INDEX ix_empleados_tienda_id (tienda_id),
  INDEX ix_empleados_activo    (activo),
  INDEX ix_empleados_nombre    (apellido, nombre)

) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;
