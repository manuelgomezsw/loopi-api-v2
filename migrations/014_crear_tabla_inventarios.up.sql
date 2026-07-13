CREATE TABLE inventarios (
  id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  tienda_id       BIGINT UNSIGNED  NOT NULL,
  fecha           DATE             NOT NULL,
  tipo            ENUM('diario','semanal','mensual','inicial') NOT NULL,
  horario         ENUM('apertura','mediodia','cierre') NULL,
  horario_norm    VARCHAR(10)      GENERATED ALWAYS AS (COALESCE(horario, '')) STORED,
  estado          ENUM('en_progreso','completado') NOT NULL DEFAULT 'en_progreso',
  responsable_id  BIGINT UNSIGNED  NOT NULL,
  iniciado_en     DATETIME         NOT NULL,
  completado_en   DATETIME         NULL,
  creado_en       DATETIME         NOT NULL,
  actualizado_en  DATETIME         NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT uq_inventarios_tienda_tipo_horario_fecha
    UNIQUE (tienda_id, tipo, horario_norm, fecha),
  CONSTRAINT fk_inventarios_tienda FOREIGN KEY (tienda_id) REFERENCES tiendas (id),
  CONSTRAINT fk_inventarios_responsable FOREIGN KEY (responsable_id) REFERENCES empleados (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX ix_inventarios_tienda_fecha ON inventarios (tienda_id, fecha DESC);
CREATE INDEX ix_inventarios_tienda_tipo_estado_completado ON inventarios (tienda_id, tipo, estado, completado_en DESC);
CREATE INDEX ix_inventarios_tienda_estado_completado ON inventarios (tienda_id, estado, completado_en DESC);
