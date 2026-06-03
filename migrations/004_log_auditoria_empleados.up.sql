-- Migración 004: Crear tabla log_auditoria_empleados
-- Dependencia: migración 003 (tabla empleados) debe estar aplicada
-- INMUTABLE: no conceder UPDATE/DELETE al usuario de la aplicación

SET time_zone = 'America/Bogota';

CREATE TABLE IF NOT EXISTS log_auditoria_empleados (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  actor_id    BIGINT UNSIGNED NOT NULL
    COMMENT 'Empleado admin que ejecutó la acción',
  accion      ENUM('CREAR', 'EDITAR', 'INACTIVAR', 'REACTIVAR', 'RESET_CONTRASENA') NOT NULL,
  empleado_id BIGINT UNSIGNED NOT NULL
    COMMENT 'Empleado afectado',
  detalle     JSON            NULL
    COMMENT 'Campos anteriores/nuevos según acción; nunca incluye contraseñas',
  creado_en   DATETIME        NOT NULL,

  PRIMARY KEY (id),
  CONSTRAINT fk_log_audit_actor    FOREIGN KEY (actor_id)    REFERENCES empleados(id),
  CONSTRAINT fk_log_audit_empleado FOREIGN KEY (empleado_id) REFERENCES empleados(id),

  INDEX ix_log_audit_empleado_id (empleado_id, creado_en),
  INDEX ix_log_audit_actor_id    (actor_id,    creado_en)

) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;
