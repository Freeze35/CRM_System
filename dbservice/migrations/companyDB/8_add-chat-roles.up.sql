-- Таблица для действий, доступных ролям
CREATE TABLE IF NOT EXISTS available_actions_chat
(
    id           SERIAL PRIMARY KEY,     -- Уникальный идентификатор действия
    role_id      INTEGER NOT NULL,       -- Внешний ключ для идентификатора роли
    create_role  BOOLEAN DEFAULT FALSE,  -- Определяет возможность создания новых ролей
    UNIQUE(role_id),        -- Уникальная комбинация роли и действия
    FOREIGN KEY (role_id) REFERENCES chat_roles(id) ON DELETE CASCADE -- Связь с таблицей chat_roles
);