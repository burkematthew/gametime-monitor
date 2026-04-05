DROP TRIGGER IF EXISTS trg_games_updated_at ON games;
DROP TRIGGER IF EXISTS trg_games_created_at ON games;
DROP FUNCTION IF EXISTS set_updated_at();
DROP FUNCTION IF EXISTS set_created_at();
DROP TABLE IF EXISTS games;
