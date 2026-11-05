BEGIN;

CREATE INDEX idx_conversation_messages_user_created_at
    ON conversation_messages (created_at) INCLUDE (jid)
    WHERE role = 'user';

COMMIT;