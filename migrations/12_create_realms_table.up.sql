CREATE TABLE IF NOT EXISTS realms (
    team TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    public_data BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO realms (team, name, public_data) VALUES ('frc2733', 'Pigmice', TRUE);
