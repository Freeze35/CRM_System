CREATE TABLE IF NOT EXISTS availableactions
(
    id SERIAL PRIMARY KEY,               -- Поле id с автоинкрементом (первичный ключ)
    roleId INT NOT NULL,                 -- Поле roleId, не допускает NULL, внешний ключ
    createTasks BOOLEAN DEFAULT FALSE,   -- Поле createTasks, булевое
    createChats BOOLEAN DEFAULT FALSE,   -- Поле createChats, булевое
    addWorkers BOOLEAN DEFAULT FALSE,    -- Поле addWorkers, булевое
    CONSTRAINT fk_role FOREIGN KEY (roleId) REFERENCES rights(id) ON DELETE CASCADE -- Внешний ключ на таблицу Rights
);
