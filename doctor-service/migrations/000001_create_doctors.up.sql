CREATE TABLE IF NOT EXISTS doctors (
    id UUID PRIMARY KEY,
    full_name TEXT NOT NULL,
    specialization TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
