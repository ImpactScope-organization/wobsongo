BEGIN;

UPDATE claims SET platform = 'whatsapp'
WHERE format = 'video' AND platform = 'tiktok';

COMMIT;