CREATE TABLE stars (
    user_id INTEGER REFERENCES users(id),
    event_key TEXT REFERENCES events(key)
)