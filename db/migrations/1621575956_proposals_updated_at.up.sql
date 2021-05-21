ALTER TABLE proposals
    ADD COLUMN updated_at timestamp;

UPDATE proposals
SET updated_at = now();

ALTER TABLE proposals
    ALTER COLUMN updated_at SET NOT NULL;
