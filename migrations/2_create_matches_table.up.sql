CREATE TABLE IF NOT EXISTS matches (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events,
    predicted_time TIMESTAMPTZ,
	actual_time TIMESTAMPTZ,
	red_score INTEGER,
	blue_score INTEGER
)
