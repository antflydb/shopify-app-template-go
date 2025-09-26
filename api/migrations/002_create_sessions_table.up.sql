-- Create sessions table
CREATE TABLE sessions (
    session_id VARCHAR(255) PRIMARY KEY,
    store_id VARCHAR(255) NOT NULL
);

-- Create index for session lookups
CREATE INDEX idx_sessions_store_id ON sessions (store_id);