CREATE TABLE
  IF NOT EXISTS players (
    id INTEGER PRIMARY KEY UNIQUE,
    name TEXT,
    team_id INT,
    FOREIGN KEY (team_id) REFERENCES teams (id)
  );