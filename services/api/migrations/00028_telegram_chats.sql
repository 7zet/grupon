CREATE TABLE telegram_chats (
  chat_id bigint PRIMARY KEY,
  language text NOT NULL DEFAULT 'uz' CHECK (language IN ('uz', 'ru', 'en')),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- migrate:down
DROP TABLE telegram_chats;
