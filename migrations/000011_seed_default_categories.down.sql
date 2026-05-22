UPDATE categories
SET is_active = FALSE
WHERE slug IN ('tax', 'audit', 'bookkeeping');
