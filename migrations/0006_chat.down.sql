-- Reverse of 0006_chat. Drop messages first (FK → chat_threads), then threads.
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_threads;
