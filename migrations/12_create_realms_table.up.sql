CREATE TABLE IF NOT EXISTS realms (
    team TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    share_reports BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO realms (team, name, share_reports) VALUES ('frc2733', 'Pigmice', TRUE);
