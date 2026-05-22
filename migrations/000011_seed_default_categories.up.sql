INSERT INTO categories(slug, name, is_active)
VALUES
    ('tax', 'Tax Services', TRUE),
    ('audit', 'Audit', TRUE),
    ('bookkeeping', 'Bookkeeping', TRUE)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    is_active = TRUE;
