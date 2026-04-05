CREATE TABLE IF NOT EXISTS games (
    id SERIAL PRIMARY KEY,
    spreadsheet_id TEXT NOT NULL,
    tournament_date DATE NOT NULL,
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    game_time TIMESTAMPTZ NOT NULL,
    location_id INTEGER REFERENCES locations(id),
    field_number TEXT,
    home_score INTEGER,
    away_score INTEGER,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(home_team, away_team, tournament_date)
);

CREATE OR REPLACE FUNCTION set_created_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.created_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_games_created_at
    BEFORE INSERT ON games
    FOR EACH ROW EXECUTE FUNCTION set_created_at();

CREATE TRIGGER trg_games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
