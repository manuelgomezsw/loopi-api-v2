CREATE TABLE categorias (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  nombre           VARCHAR(100)    NOT NULL
                     COMMENT 'Nombre único en el sistema (case-insensitive por utf8mb4_unicode_ci)',
  activo           TINYINT(1)      NOT NULL DEFAULT 1,
  creado_por       BIGINT UNSIGNED NOT NULL COMMENT 'FK a usuarios.id — quién creó el registro',
  creado_en        DATETIME        NOT NULL,
  actualizado_por  BIGINT UNSIGNED NOT NULL COMMENT 'FK a usuarios.id — último que modificó',
  actualizado_en   DATETIME        NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_categorias_nombre
    UNIQUE (nombre),
  CONSTRAINT fk_categorias_creado_por
    FOREIGN KEY (creado_por) REFERENCES usuarios (id),
  CONSTRAINT fk_categorias_actualizado_por
    FOREIGN KEY (actualizado_por) REFERENCES usuarios (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_categorias_activo ON categorias (activo);
