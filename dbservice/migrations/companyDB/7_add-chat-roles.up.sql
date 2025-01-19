CREATE TABLE IF NOT EXISTS chat_roles
(
    id         SERIAL PRIMARY KEY,       -- Уникальный идентификатор роли
    chat_id    INTEGER NOT NULL,         -- Внешний ключ для идентификатора чата
    name_role  VARCHAR(100) NOT NULL,    -- Название роли (например, admin_chat, user_chat)
    removable  BOOLEAN DEFAULT FALSE,    -- Определяет, можно ли удалить роль
    UNIQUE(chat_id, name_role),          -- Уникальная комбинация чата и названия роли
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE -- Связь с таблицей chats
);