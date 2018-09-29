CREATE TABLE IF NOT EXISTS events (
	key TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	district TEXT,
    week INTEGER,
	start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    location_name TEXT NOT NULL,
    lat REAL NOT NULL,
    lon REAL NOT NULL
)
