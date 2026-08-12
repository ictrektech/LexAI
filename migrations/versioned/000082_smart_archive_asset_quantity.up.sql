-- Migration 80 was deployed in some environments before quantity-only
-- asset support was added to the source schema. Converge those databases
-- without touching existing asset rows.
ALTER TABLE archive_assets
    ADD COLUMN IF NOT EXISTS is_quantity_only BOOLEAN NOT NULL DEFAULT FALSE;
