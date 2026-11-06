BEGIN;

ALTER TABLE topics ADD CONSTRAINT topics_name_key UNIQUE (name);
DROP INDEX IF EXISTS uq_topics_lower_name;

ALTER TABLE influencers
    ADD CONSTRAINT influencers_platform_username_key UNIQUE (platform, username);
DROP INDEX IF EXISTS uq_influencers_platform_lower_username;

COMMIT;