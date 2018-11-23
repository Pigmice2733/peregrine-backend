CREATE TABLE IF NOT EXISTS schema_overrides (
    realm_id INTEGER NOT NULL REFERENCES realms ON DELETE CASCADE,
    schema_id INTEGER NOT NULL REFERENCES schemas ON DELETE CASCADE,

    UNIQUE(realm_id, schema_id)
);
