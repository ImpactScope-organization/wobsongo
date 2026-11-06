BEGIN;

-- Claims from TikTok links sent to the bot were stored as whatsapp.
UPDATE claims SET platform = 'tiktok'
WHERE format = 'video' AND platform = 'whatsapp';

COMMIT;