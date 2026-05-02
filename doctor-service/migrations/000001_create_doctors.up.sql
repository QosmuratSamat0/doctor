CREATE TABLE IF NOT EXISTS doctors (
    id UUID PRIMARY KEY,
    full_name TEXT NOT NULL,
    specialization TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);
