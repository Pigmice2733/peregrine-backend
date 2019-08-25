BEGIN;

UPDATE schemas SET auto = auto || teleop;

ALTER TABLE schemas RENAME auto TO schema;
ALTER TABLE schemas DROP COLUMN teleop;

ALTER TABLE reports DROP COLUMN auto_name;

COMMIT;