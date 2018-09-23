CREATE TABLE IF NOT EXISTS webcasts (
    id SERIAL PRIMARY KEY,
    eventKey TEXT NOT NULL,
    type TEXT NOT NULL,
    url TEXT NOT NULL,

    FOREIGN KEY(eventKey) REFERENCES events(key)
)