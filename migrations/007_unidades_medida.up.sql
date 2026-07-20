CREATE TABLE unidades_medida (
  id                BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  codigo            VARCHAR(20)      NOT NULL          COMMENT 'Código corto único (ej. kg, ml, und). Inmutable si hay items asignados.',
  nombre            VARCHAR(100)     NOT NULL          COMMENT 'Nombre completo (ej. Kilogramo, Mililitro)',
  tipo_medida       ENUM(
                      'peso',
                      'volumen',
                      'unidad'
                    )                NOT NULL          COMMENT 'Tipo de magnitud física. Área diferida a versión futura.',
  factor_conversion DECIMAL(12,4)    NOT NULL          COMMENT 'Unidades de la base equivalentes a 1 unidad de esta. Base: factor = 1.0000.',
  unidad_base       TINYINT(1)       NOT NULL DEFAULT 0 COMMENT '1 = unidad base del tipo (g, ml, und). Solo una por tipo.',
  activo            TINYINT(1)       NOT NULL DEFAULT 1,
  creado_en         DATETIME         NOT NULL,
  actualizado_en    DATETIME         NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_unidades_medida_codigo UNIQUE (codigo),
  CONSTRAINT chk_unidades_medida_factor CHECK (factor_conversion > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_unidades_medida_tipo_activo ON unidades_medida (tipo_medida, activo);
