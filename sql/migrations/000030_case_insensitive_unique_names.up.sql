BEGIN;

-- Influencer usernames: TikTok treats @SomeOne and @someone as one account.
CREATE UNIQUE INDEX uq_influencers_platform_lower_username
    ON influencers (platform, lower(username));
ALTER TABLE influencers
    DROP CONSTRAINT IF EXISTS influencers_platform_username_key;

-- Topic names: "Health" and "health" must not both exist.
CREATE UNIQUE INDEX uq_topics_lower_name ON topics (lower(name));
ALTER TABLE topics DROP CONSTRAINT IF EXISTS topics_name_key;

COMMIT;