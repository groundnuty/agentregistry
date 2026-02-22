-- Add A2A Agent Card column to agents table.
-- Stores the Agent Card JSON blob as opaque JSONB to accommodate both
-- camelCase (Go SDK) and snake_case (Python SDK) field conventions.

ALTER TABLE agents ADD COLUMN IF NOT EXISTS a2a_card JSONB;

-- Partial index for efficiently filtering agents that have an Agent Card
CREATE INDEX IF NOT EXISTS idx_agents_has_a2a_card
    ON agents ((a2a_card IS NOT NULL)) WHERE a2a_card IS NOT NULL;

-- GIN index for JSONB containment queries (e.g., skill tag filtering)
CREATE INDEX IF NOT EXISTS idx_agents_a2a_card_gin
    ON agents USING GIN (a2a_card jsonb_path_ops)
    WHERE a2a_card IS NOT NULL;
