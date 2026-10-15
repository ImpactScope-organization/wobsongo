BEGIN;

ALTER TABLE users DROP COLUMN IF EXISTS password_changed_at;

COMMIT;
