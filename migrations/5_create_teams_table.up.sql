CREATE TABLE IF NOT EXISTS teams (
    key TEXT NOT NULL,
    event_key TEXT NOT NULL REFERENCES events,
    rank INTEGER,
    ranking_score REAL,

    PRIMARY KEY(key, event_key)
)
