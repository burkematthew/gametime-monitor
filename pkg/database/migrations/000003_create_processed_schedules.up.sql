CREATE TABLE IF NOT EXISTS processed_schedules (
    spreadsheet_id TEXT PRIMARY KEY,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
