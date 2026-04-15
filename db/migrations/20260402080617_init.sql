-- +goose Up
-- +goose StatementBegin
CREATE TABLE links (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  original_url TEXT NOT NULL,
  short_name TEXT UNIQUE NOT NULL,
  short_url TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NUll DEFAULT NOW()
);

CREATE TABLE link_visits (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  link_id BIGINT NOT NULL REFERENCES links(id),
  ip TEXT NOT NULL,
  user_agent TEXT NOT NULL,
  status INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS links;

DROP TABLE IF EXISTS link_visits;
-- +goose StatementEnd
