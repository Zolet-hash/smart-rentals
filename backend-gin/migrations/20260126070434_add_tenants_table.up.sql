CREATE TABLE tenants (
    id              SERIAL PRIMARY KEY,
    tenant_name     VARCHAR(255) NOT NULL,
    payment_no      VARCHAR(100),
    unit_id         INTEGER NOT NULL,
    landlord_id     INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_tenants_unit
        FOREIGN KEY (unit_id)
        REFERENCES units (id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_tenant_landlord
        FOREIGN KEY (landlord_id)
        REFERENCES users (id)
        ON DELETE RESTRICT
);
