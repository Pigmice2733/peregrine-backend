CREATE TABLE IF NOT EXISTS webcasts (
    event_key TEXT NOT NULL REFERENCES events,
    type TEXT NOT NULL,
    url TEXT NOT NULL
)
