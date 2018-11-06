ALTER TABLE alliances
    DROP CONSTRAINT alliances_match_key_fkey,
    ADD CONSTRAINT alliances_match_key_fkey
        FOREIGN KEY (match_key)
        REFERENCES matches (key);
