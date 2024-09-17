CREATE TABLE IF NOT EXISTS authUsers
(
    id        SERIAL PRIMARY KEY,        -- Первичный ключ (id)
    email     VARCHAR(100) NOT NULL UNIQUE,  -- Электронная почта пользователя, уникальная
    phone     VARCHAR(50)  NOT NULL UNIQUE,  -- Телефон пользователя, уникальный
    password  VARCHAR(100) NOT NULL,     -- Пароль пользователя
    companyId INT NOT NULL,              -- Ссылка на колонку id в таблице companies
    createdAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- Время создания
    FOREIGN KEY (companyId) REFERENCES companies(id)  -- Внешний ключ, связывающий с таблицей companies
);
