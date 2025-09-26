-- Create sessions table
CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL
);

-- Create index for session lookups
CREATE INDEX idx_sessions_store_id ON sessions (store_id);