CREATE TABLE IF NOT EXISTS chat_users
(
    user_id INT NOT NULL,
    chat_id INT NOT NULL,
    PRIMARY KEY (user_id, chat_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
    );