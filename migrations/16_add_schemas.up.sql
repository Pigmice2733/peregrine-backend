CREATE TABLE IF NOT EXISTS schemas (
    id SERIAL PRIMARY KEY,
    year INTEGER UNIQUE,
    realm_id INTEGER REFERENCES realms,
    auto JSONB,
    teleop JSONB
);

INSERT INTO schemas (auto, teleop) VALUES ('[]', '[]');

ALTER TABLE events
    ADD COLUMN schema_id INTEGER REFERENCES schemas DEFAULT 1;
