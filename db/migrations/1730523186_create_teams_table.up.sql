CREATE TABLE
  IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY UNIQUE,
    name TEXT NOT NULL UNIQUE,
    city TEXT,
    abbreviation TEXT,
    conference TEXT,
    division TEXT,
    code TEXT,
    slug TEXT,
    min_year INT,
    max_year INT
  );