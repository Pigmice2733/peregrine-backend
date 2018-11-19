CREATE TABLE IF NOT EXISTS realms (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    share_reports BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO realms (name, share_reports) VALUES ('Pigmice', FALSE);
