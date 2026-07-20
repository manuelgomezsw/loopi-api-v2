CREATE TABLE items (
  id                    BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  codigo                VARCHAR(20)      NOT NULL
                          COMMENT 'Código único asignado por el admin (ej. CAF-001). Case-sensitive.',
  nombre                VARCHAR(150)     NOT NULL
                          COMMENT 'Nombre único en el sistema (case-insensitive por utf8mb4_unicode_ci).',
  tipo                  ENUM(
                          'insumo',
                          'material_consumo',
                          'activo'
                        )                NOT NULL,
  subcategoria_id       BIGINT UNSIGNED  NOT NULL
                          COMMENT 'FK a subcategorias.id. Puede cambiarse libremente.',
  proveedor_id          BIGINT UNSIGNED  NULL
                          COMMENT 'FK a proveedores.id. NULL = sin proveedor asignado.',
  unidad_medida_id      BIGINT UNSIGNED  NOT NULL
                          COMMENT 'FK a unidades_medida.id. Cambio con historial requiere confirmación.',
  costo_unitario        INT              NULL
                          COMMENT 'Costo por defecto global en COP sin decimales. NULL si no definido.',
  frecuencia_inventario ENUM(
                          'diario',
                          'semanal',
                          'mensual'
                        )                NOT NULL,
  stock_seguridad       DECIMAL(12,4)    NOT NULL
                          COMMENT 'En la unidad de medida del item. 0 si no aplica (p. ej. activos).',
  tiempo_entrega_dias   SMALLINT UNSIGNED NULL
                          COMMENT 'Días de entrega del proveedor habitual. NULL si no definido.',
  activo                TINYINT(1)       NOT NULL DEFAULT 1,
  creado_por            BIGINT UNSIGNED  NOT NULL
                          COMMENT 'FK a empleados.id — quién creó el item.',
  creado_en             DATETIME         NOT NULL,
  actualizado_por       BIGINT UNSIGNED  NOT NULL
                          COMMENT 'FK a empleados.id — último que modificó el item.',
  actualizado_en        DATETIME         NOT NULL,

  PRIMARY KEY (id),

  CONSTRAINT uq_items_codigo
    UNIQUE (codigo),
  CONSTRAINT uq_items_nombre
    UNIQUE (nombre),

  CONSTRAINT fk_items_subcategoria
    FOREIGN KEY (subcategoria_id) REFERENCES subcategorias (id),
  CONSTRAINT fk_items_proveedor
    FOREIGN KEY (proveedor_id) REFERENCES proveedores (id),
  CONSTRAINT fk_items_unidad_medida
    FOREIGN KEY (unidad_medida_id) REFERENCES unidades_medida (id),
  CONSTRAINT fk_items_creado_por
    FOREIGN KEY (creado_por) REFERENCES empleados (id),
  CONSTRAINT fk_items_actualizado_por
    FOREIGN KEY (actualizado_por) REFERENCES empleados (id)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_items_activo ON items (activo);
CREATE INDEX ix_items_tipo_activo ON items (tipo, activo);
CREATE INDEX ix_items_frecuencia_activo ON items (frecuencia_inventario, activo);
