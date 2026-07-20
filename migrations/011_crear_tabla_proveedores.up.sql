CREATE TABLE proveedores (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  razon_social       VARCHAR(255)    NOT NULL COMMENT 'Nombre legal del proveedor.',
  nit                VARCHAR(50)     NOT NULL COMMENT 'Identificador único. Cadena libre; acepta NITs formales y códigos internos.',
  nombre_contacto    VARCHAR(150)    NOT NULL COMMENT 'Nombre de la persona de contacto. Obligatorio.',
  telefono_contacto  VARCHAR(50)     NOT NULL COMMENT 'Teléfono de contacto. Obligatorio.',
  email_contacto     VARCHAR(255)    NULL     COMMENT 'Email de contacto. Opcional.',
  activo             TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '1 = activo, 0 = inactivo (soft delete).',
  creado_en          DATETIME        NOT NULL,
  actualizado_en     DATETIME        NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_proveedores_nit
    UNIQUE (nit)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_proveedores_activo ON proveedores (activo);
CREATE INDEX ix_proveedores_razon_social ON proveedores (razon_social);
