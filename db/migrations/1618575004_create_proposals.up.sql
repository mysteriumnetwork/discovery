CREATE TABLE IF NOT EXISTS proposals
(
    key        text PRIMARY KEY,
    proposal   jsonb     NOT NULL,
    expires_at timestamp NOT NULL
);
