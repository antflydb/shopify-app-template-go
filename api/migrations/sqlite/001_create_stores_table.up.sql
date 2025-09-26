-- Create stores table
CREATE TABLE stores (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,
    nonce TEXT,
    access_token TEXT,
    installed INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    deleted_at DATETIME
);

-- Create indexes
CREATE INDEX idx_stores_deleted_at ON stores (deleted_at);
CREATE INDEX idx_stores_name ON stores (name);