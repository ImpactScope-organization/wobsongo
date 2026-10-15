BEGIN;

-- Emails are compared lowercased and trimmed from now on, so existing rows
-- must match. Fails (and rolls back) if two accounts differ only by case.
UPDATE users SET email = lower(btrim(email));

-- When the password was last set. Refresh tokens issued before it are
-- rejected, which signs out other devices after a password change.
ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ;
UPDATE users SET password_changed_at = created_at;
ALTER TABLE users ALTER COLUMN password_changed_at SET NOT NULL;
ALTER TABLE users ALTER COLUMN password_changed_at SET DEFAULT NOW();

COMMIT;
