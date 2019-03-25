CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    event_key TEXT NOT NULL REFERENCES events,
    match_key TEXT NOT NULL REFERENCES matches,
    team_key TEXT NOT NULL,
    reporter_id INTEGER REFERENCES users ON DELETE SET NULL,
    realm_id INTEGER REFERENCES realms ON DELETE SET NULL,
    comment TEXT NOT NULL,

    UNIQUE(event_key, match_key, team_key, reporter_id)
);
