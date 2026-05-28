CREATE TYPE order_report_status AS ENUM ('pending', 'dismissed', 'order_removed');

CREATE TABLE order_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL CHECK (char_length(trim(reason)) >= 10),
    status order_report_status NOT NULL DEFAULT 'pending',
    admin_id UUID REFERENCES users(id) ON DELETE SET NULL,
    admin_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_order_reports_status_created ON order_reports(status, created_at DESC);
CREATE INDEX idx_order_reports_order_id ON order_reports(order_id);
CREATE INDEX idx_order_reports_reporter_id ON order_reports(reporter_id);

CREATE UNIQUE INDEX idx_order_reports_pending_per_reporter
    ON order_reports(order_id, reporter_id)
    WHERE status = 'pending';
