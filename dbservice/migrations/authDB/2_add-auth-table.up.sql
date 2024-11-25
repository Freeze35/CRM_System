CREATE TABLE IF NOT EXISTS authUsers
(
    id        SERIAL PRIMARY KEY,               -- Первичный ключ (id)
    email     VARCHAR(100) NOT NULL UNIQUE,     -- Электронная почта пользователя, уникальная
    phone     VARCHAR(50)  NOT NULL UNIQUE,     -- Телефон пользователя, уникальный
    password  VARCHAR(100) NOT NULL,            -- Пароль пользователя
    company_id INT NOT NULL,                     -- Ссылка на колонку id в таблице companies
    createdAt TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Время создания в формате UTC
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE -- Внешний ключ
    );
