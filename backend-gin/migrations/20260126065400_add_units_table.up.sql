CREATE TABLE IF NOT EXISTS units (
    id              SERIAL PRIMARY KEY,
    property_id     INTEGER NOT NULL,
    unit_name       VARCHAR(100) NOT NULL,
    unit_type       VARCHAR(100),
    unit_price      INTEGER NOT NULL,
    vacancy         BOOLEAN,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_units_property
        FOREIGN KEY (property_id)
        REFERENCES properties (id)
        ON DELETE CASCADE
);