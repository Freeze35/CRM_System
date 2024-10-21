CREATE TABLE IF NOT EXISTS messages
(
    id          SERIAL PRIMARY KEY,         -- Уникальный идентификатор сообщения
    chat_id     INT NOT NULL,               -- Внешний ключ на таблицу chats
    user_id     INT NOT NULL,               -- Внешний ключ на таблицу users (автор сообщения)
    message     TEXT NOT NULL,              -- Текст сообщения
    created_at  TIMESTAMP DEFAULT NOW(),    -- Время отправки сообщения
    CONSTRAINT fk_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,  -- Внешний ключ на таблицу chats
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL   -- Внешний ключ на таблицу users в случае удаления одного из пользователей останутся сообщения
    );
