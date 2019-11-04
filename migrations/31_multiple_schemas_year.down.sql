BEGIN;
DELETE FROM schemas a USING schemas b WHERE a.id > b.id AND a.year = b.year;
ALTER TABLE schemas ADD CONSTRAINT schemas_year_key UNIQUE(year);
COMMIT;