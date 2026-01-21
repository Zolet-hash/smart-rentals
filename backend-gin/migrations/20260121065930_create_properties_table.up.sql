CREATE TABLE IF NOT EXISTS properties (
    id BIGSERIAL PRIMARY KEY,
    landlord_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    location VARCHAR(255),
    property_type VARCHAR(100),
    vacancy BOOLEAN,
    total_rent INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_properties_landlord
        FOREIGN KEY (landlord_id) REFERENCES users(id)
);
