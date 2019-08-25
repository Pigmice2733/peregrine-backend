ALTER TABLE matches
    ADD COLUMN red_score_breakdown JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN blue_score_breakdown JSONB NOT NULL DEFAULT '{}';