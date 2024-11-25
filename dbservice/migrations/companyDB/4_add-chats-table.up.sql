CREATE TABLE IF NOT EXISTS chats
(
    id          SERIAL PRIMARY KEY,      -- Уникальный идентификатор чата
    chat_name   VARCHAR(255) NOT NULL,   -- Название чата
    created_at  TIMESTAMPTZ DEFAULT NOW()  -- Время создания чата
    );