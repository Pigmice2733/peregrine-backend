BEGIN;
ALTER TABLE stars DROP CONSTRAINT stars_event_key_fkey;
ALTER TABLE stars ADD CONSTRAINT stars_event_key_fkey FOREIGN KEY(user_id) REFERENCES users(id);
COMMIT;