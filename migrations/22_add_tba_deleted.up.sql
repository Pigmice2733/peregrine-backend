ALTER TABLE events ADD COLUMN tba_deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE matches ADD COLUMN tba_deleted BOOLEAN NOT NULL DEFAULT false;