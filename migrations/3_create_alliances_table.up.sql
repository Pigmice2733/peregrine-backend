CREATE TABLE IF NOT EXISTS alliances (
    team_keys TEXT[3] NOT NULL,
    match_id TEXT NOT NULL REFERENCES matches,
    is_blue BOOLEAN NOT NULL,

    PRIMARY KEY(match_id, is_blue)
)
