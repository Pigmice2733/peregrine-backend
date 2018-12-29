ALTER TABLE events
    DROP CONSTRAINT events_schema_id_fkey,
    ADD CONSTRAINT events_schema_id_fkey
        FOREIGN KEY (schema_id)
        REFERENCES schemas (id);
