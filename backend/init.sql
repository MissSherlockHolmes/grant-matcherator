CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS profiles (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    bio TEXT,
    interests TEXT[] DEFAULT '{}',
    location VARCHAR(255),
    looking_for TEXT,
    age INTEGER,
    occupation VARCHAR(255),
    profile_picture_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS matches (
    id SERIAL PRIMARY KEY,
    user_id_1 INTEGER REFERENCES users(id),
    user_id_2 INTEGER REFERENCES users(id),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id_1, user_id_2)
);

CREATE TABLE IF NOT EXISTS match_preferences (
    user_id INT PRIMARY KEY REFERENCES users(id),
    priority TEXT CHECK (priority IN ('looking_for', 'interests', 'age', 'none')) DEFAULT 'none'
);


CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    match_id INTEGER REFERENCES matches(id),
    sender_id INTEGER REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id VARCHAR(255) PRIMARY KEY,
    match_id INTEGER REFERENCES matches(id),
    sender_id INTEGER REFERENCES users(id),
    content TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    read BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    token VARCHAR(500) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS user_status (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    last_active TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_status CHECK (status IN ('online', 'offline'))
);

-- Add necessary indexes
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_matches_users ON matches(user_id_1, user_id_2);
CREATE INDEX IF NOT EXISTS idx_messages_match ON messages(match_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_match ON chat_messages(match_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_sender ON chat_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_tokens_user ON tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_tokens_token ON tokens(token);
CREATE INDEX IF NOT EXISTS idx_user_status_user ON user_status(user_id);
ALTER TABLE user_status
ADD COLUMN last_match_check TIMESTAMP DEFAULT NULL,
ADD COLUMN last_message_check TIMESTAMP DEFAULT NULL;


-- Add trigger to auto-insert user_status entry when a new user is created
CREATE OR REPLACE FUNCTION create_user_status_trigger()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_status (user_id, status, last_active)
    VALUES (NEW.id, 'offline', CURRENT_TIMESTAMP)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_status_trigger
AFTER INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION create_user_status_trigger();
