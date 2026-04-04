-- +goose Up
CREATE TABLE links (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  original_url TEXT NOT NULL,
  short_name TEXT UNIQUE NOT NULL,
  short_url TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);


-- +goose Down
DROP TABLE IF EXISTS links;
