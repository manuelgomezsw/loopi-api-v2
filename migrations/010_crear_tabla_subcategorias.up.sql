CREATE TABLE subcategorias (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  nombre           VARCHAR(100)    NOT NULL
                     COMMENT 'Único dentro de la categoría, case-insensitive',
  categoria_id     BIGINT UNSIGNED NOT NULL
                     COMMENT 'FK a categorias.id — inmutable tras creación',
  activo           TINYINT(1)      NOT NULL DEFAULT 1,
  creado_por       BIGINT UNSIGNED NOT NULL COMMENT 'FK a usuarios.id',
  creado_en        DATETIME        NOT NULL,
  actualizado_por  BIGINT UNSIGNED NOT NULL COMMENT 'FK a usuarios.id',
  actualizado_en   DATETIME        NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_subcategorias_nombre_categoria
    UNIQUE (categoria_id, nombre),
  CONSTRAINT fk_subcategorias_categoria
    FOREIGN KEY (categoria_id) REFERENCES categorias (id),
  CONSTRAINT fk_subcategorias_creado_por
    FOREIGN KEY (creado_por) REFERENCES usuarios (id),
  CONSTRAINT fk_subcategorias_actualizado_por
    FOREIGN KEY (actualizado_por) REFERENCES usuarios (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_subcategorias_categoria_activo ON subcategorias (categoria_id, activo);
