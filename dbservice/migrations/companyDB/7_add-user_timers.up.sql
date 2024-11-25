CREATE TABLE user_timers
(
    id          SERIAL PRIMARY KEY,             -- Уникальный идентификатор записи
    user_id     INTEGER   NOT NULL,             -- Идентификатор пользователя (внешний ключ на таблицу пользователей)
    start_time  TIMESTAMPTZ NOT NULL,             -- Время начала отсчёта таймера
    end_time    TIMESTAMPTZ,                      -- Время окончания отсчёта таймера (может быть NULL для активных таймеров)
    duration INTERVAL GENERATED ALWAYS AS       -- Продолжительность (рассчитывается на основе start_time и end_time)
        (end_time - start_time) STORED,         -- STORED сохраняет результат вычисления в базе данных, а не рассчитывает его каждый раз при обращении
    description TEXT,                           -- Описание или комментарий к таймеру
    is_active   BOOLEAN   DEFAULT TRUE,         -- Флаг, указывающий, активен ли таймер (актуально, если end_time NULL)
    created_at  TIMESTAMPTZ DEFAULT NOW(),        -- Время создания записи
    updated_at  TIMESTAMPTZ DEFAULT NOW(),        -- Время последнего обновления записи
    FOREIGN KEY (user_id) REFERENCES users (id) -- Связь с таблицей пользователей
);