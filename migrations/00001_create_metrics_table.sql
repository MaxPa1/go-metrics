-- +goose Up
CREATE TABLE metrics (
    id VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    delta BIGINT,
    value double precision,
    hash  VARCHAR(255),
    PRIMARY KEY (id, type)
);

-- +goose Down
DROP TABLE IF EXISTS metrics;
