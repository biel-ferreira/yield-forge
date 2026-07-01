-- 0006_chat — conversational copilot threads + messages (SPEC-108 FR-1086).
--
-- Two tables: a per-user conversation thread and its ordered messages. Follows the 0001–0005
-- conventions: UUID PKs, timestamptz/UTC, snake_case, FK ON DELETE CASCADE. No money columns —
-- amounts live only inside the transient grounding facts, never persisted (BR-1084). user_id and
-- thread_id are indexed for the per-user list/scope queries and the message-by-thread reads;
-- mutations are additionally scoped by (id, user_id) at the app layer (BR-1083). Storage is bounded
-- and clearable at the app layer (rolling eviction), so no unbounded growth (BR-1085).

CREATE TABLE chat_threads (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      text        NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL          -- advances per turn (rolling-eviction order)
);
CREATE INDEX chat_threads_user_id_idx ON chat_threads (user_id);

CREATE TABLE chat_messages (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id   uuid        NOT NULL REFERENCES chat_threads (id) ON DELETE CASCADE,
    role        text        NOT NULL,        -- 'user' | 'assistant'
    content     text        NOT NULL,
    explanation text        NULL,            -- assistant only (explainability-gate guarantee)
    created_at  timestamptz NOT NULL         -- ordering within the thread
);
CREATE INDEX chat_messages_thread_id_idx ON chat_messages (thread_id);
