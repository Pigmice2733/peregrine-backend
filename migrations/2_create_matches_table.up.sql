CREATE TABLE IF NOT EXISTS matches (
    key TEXT PRIMARY KEY,
    event_key TEXT NOT NULL REFERENCES events,
    predicted_time TIMESTAMPTZ,
	actual_time TIMESTAMPTZ,
	red_score INTEGER,
	blue_score INTEGER
)
