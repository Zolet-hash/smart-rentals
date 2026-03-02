--created
CREATE TABLE users (
    id              SERIAL PRIMARY KEY,
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    phone           VARCHAR(50),
    role            VARCHAR(50) NOT NULL,
    trial_starts_at TIMESTAMPTZ NOT NULL,
    trial_ends_at   TIMESTAMPTZ NOT NULL,
    subscription_status VARCHAR(20) NOT NULL DEFAULT 'trial',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Created
CREATE TABLE properties (
    id              SERIAL PRIMARY KEY,
    landlord_id     INTEGER NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    location        VARCHAR(255) NOT NULL,
    property_type   VARCHAR(100),
    vacancy         BOOLEAN NOT NULL DEFAULT TRUE,
    total_rent      NUMERIC(12,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_properties_landlord
        FOREIGN KEY (landlord_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

CREATE TABLE units (
    id              SERIAL PRIMARY KEY,
    property_id     INTEGER NOT NULL,
    unit_name       VARCHAR(100) NOT NULL,
    unit_type       VARCHAR(100),
    unit_price      NUMERIC(12, 2),
    vacancy         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_units_property
        FOREIGN KEY (property_id)
        REFERENCES properties (id)
        ON DELETE CASCADE
);


CREATE TABLE tenants (
    id              SERIAL PRIMARY KEY,
    tenant_name     VARCHAR(255) NOT NULL,
    payment_no1      VARCHAR(100), --addition 
    payment_no2     VARCHAR(100),
    unit_id         INTEGER NOT NULL,
    landlord_id     INTEGER NOT NULL,
    rent            NUMERIC(12,2),
    balance         NUMERIC(12, 2),
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

--postgres function to set balance as current rent, why I love postgres
--now if tenant pays through cash, we can as well update balance
CREATE OR REPLACE FUNCTION set_default_balance()
RETURNS trigger AS $$
BEGIN
    IF NEW.balance IS NULL THEN
        NEW.balance := NEW.rent;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_set_default_balance
BEFORE INSERT ON tenants
FOR EACH ROW
EXECUTE FUNCTION set_default_balance();


--incase a landlord uses till
CREATE TABLE tills (
    id              BIGSERIAL PRIMARY KEY,
    till_number     VARCHAR(100) NOT NULL,
    callback_registered BOOLEAN DEFAULT false,
    landlord_id     BIGINT NOT NULL REFERENCES users(id),
    daraja_app_id BIGINT REFERENCES daraja_apps(id),
    active          BOOLEAN DEFAULT true
);
CREATE UNIQUE INDEX uniq_till_number
ON tills (till_number);

CREATE TABLE paybills (
    id              BIGSERIAL PRIMARY KEY,
    paybill         VARCHAR(100) NOT NULL, --
    callback_registered BOOLEAN DEFAULT false,
    account_number  VARCHAR(100) NOT NULL, --VARCHAR
    landlord_id     BIGINT NOT NULL REFERENCES users(id),
    daraja_app_id   BIGINT REFERENCES daraja_apps(id),
    active          BOOLEAN DEFAULT true
);
CREATE UNIQUE INDEX uniq_paybill
ON paybills (paybill)

CREATE TABLE till_transactions (
    id                  BIGSERIAL PRIMARY KEY,
    receipt_number      VARCHAR(100) NOT NULL,
    till_number         VARCHAR(100) NOT NULL,
    phone_number        VARCHAR(100) NOT NULL,
    amount              NUMERIC(12,2),
    trans_time          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id           INTEGER NOT NULL REFERENCES tenants(id),
    landlord_id         INTEGER NOT NULL REFERENCES users(id)
);

CREATE TABLE paybill_transactions (
    id                  BIGSERIAL PRIMARY KEY,
    receipt_number      VARCHAR(100) NOT NULL,
    paybill             VARCHAR(100) NOT NULL,
    account_number      VARCHAR(100) NOT NULL,
    phone_number        VARCHAR(100) NOT NULL,
    amount              NUMERIC(12,2),
    trans_time          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id           INTEGER NOT NULL REFERENCES tenants(id),
    landlord_id         INTEGER NOT NULL REFERENCES users(id)
);

CREATE TABLE daraja_apps (
  id BIGSERIAL PRIMARY KEY,
  landlord_id BIGINT REFERENCES users(id),
  environment TEXT CHECK (environment IN ('sandbox','production')),
  consumer_key TEXT,
  consumer_secret TEXT,
  passkey TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE send_money (
    id SERIAL PRIMARY KEY,
    landlord_id BIGINT REFERENCES users(id),
    tenant_id   BIGINT REFERENCES tenants(id),
    mpesa_code  VARCHAR(100) UNIQUE NOT NULL,
    sender_name VARCHAR(100),
    amount      NUMERIC(12, 2),
    phone       VARCHAR(50),
    matched     BOOLEAN DEFAULT false,
    applied     BOOLEAN DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE UNIQUE INDEX uniq_tenant_phone1
ON tenants (landlord_id, payment_no1)
WHERE payment_no1 IS NOT NULL;

CREATE UNIQUE INDEX uniq_tenant_phone2
ON tenants (landlord_id, payment_no2)
WHERE payment_no2 IS NOT NULL;


CREATE OR REPLACE FUNCTION match_send_money()
RETURNS trigger AS $$
BEGIN
	UPDATE send_money sm
	SET tenant_id = t.id,
	    matched = true
	FROM tenants t
	WHERE sm.id = NEW.id
	  AND sm.landlord_id = t.landlord_id
	  AND (
			sm.phone = t.payment_no1
	     OR sm.phone = t.payment_no2
	  );

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_match_send_money
AFTER INSERT ON send_money
FOR EACH ROW
EXECUTE FUNCTION match_send_money();

CREATE TABLE landlord_pay (
    id SERIAL PRIMARY KEY,
    landlord_id BEGIN REFERENCES users(id)
    payment_amount NUMERIC(12,2)
    payment_date ;
)

