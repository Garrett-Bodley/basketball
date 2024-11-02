CREATE TABLE
  IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    team_id INTEGER UNIQUE,
    team_city STRING,
    team_name STRING,
    team_abbreviation STRING,
    team_conference STRING,
    team_division STRING,
    team_code STRING,
    team_slug STRING,
    min_year INT,
    max_year INT
  );