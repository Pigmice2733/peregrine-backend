ALTER TABLE matches
    ADD COLUMN videos JSONB NOT NULL DEFAULT '[]';