-- +migrate Up

CREATE TABLE tokens_revocados (
    jti            VARCHAR(36)  NOT NULL,
    expira_en      DATETIME     NOT NULL,
    creado_en      DATETIME     NOT NULL,
    actualizado_en DATETIME     NOT NULL,
    PRIMARY KEY (jti),
    INDEX idx_tokens_revocados_expira_en (expira_en)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE usuarios
    ADD COLUMN intentos_fallidos  INT          NOT NULL DEFAULT 0    AFTER activo,
    ADD COLUMN bloqueado_hasta    DATETIME         NULL DEFAULT NULL  AFTER intentos_fallidos;

-- +migrate Down

ALTER TABLE usuarios
    DROP COLUMN bloqueado_hasta,
    DROP COLUMN intentos_fallidos;

DROP TABLE IF EXISTS tokens_revocados;
