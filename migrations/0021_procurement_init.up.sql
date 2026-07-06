-- =============================================================================
--  Procurement schema + raw tables
--  Worker mirrors Hashmicro purchase_request, purchase_order, stock_picking,
--  and purchase_order_purchase_request_rel into these tables.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS procurement;

-- Purchase Requests
-- Sumber: public.purchase_request (Hashmicro)
-- amount_total sudah dihitung Hashmicro, tidak perlu sync purchase_request_line.
CREATE TABLE IF NOT EXISTS procurement.purchase_requests (
    source_id            INTEGER       PRIMARY KEY,
    company_source_id    INTEGER       NOT NULL,
    name                 TEXT,
    state                TEXT          NOT NULL,
    date_start           DATE,
    amount_total         NUMERIC(20,2) NOT NULL DEFAULT 0,
    department_source_id INTEGER,
    branch_source_id     INTEGER,
    source_updated_at    TIMESTAMP,
    synced_at            TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pr_company ON procurement.purchase_requests (company_source_id);
CREATE INDEX IF NOT EXISTS idx_pr_state   ON procurement.purchase_requests (state);
CREATE INDEX IF NOT EXISTS idx_pr_date    ON procurement.purchase_requests (date_start);

-- Purchase Orders
-- Sumber: public.purchase_order (Hashmicro)
-- date_planned = Expected Date pada PO.
-- amount_untaxed = Total Untaxed Amount PO (Subtotal - Discount).
-- cycle_days dihitung worker dari date_approve - pr_confirm_date dalam hari.
CREATE TABLE IF NOT EXISTS procurement.purchase_orders (
    source_id                      INTEGER       PRIMARY KEY,
    company_source_id              INTEGER       NOT NULL,
    name                           TEXT          NOT NULL,
    partner_source_id              INTEGER       NOT NULL,
    state                          TEXT,
    date_order                     DATE          NOT NULL,
    date_approve                   DATE,
    date_planned                   TIMESTAMP,
    cycle_days                     NUMERIC(5,1),
    amount_untaxed                 NUMERIC(20,2) NOT NULL DEFAULT 0,
    amount_total                   NUMERIC(20,2) NOT NULL DEFAULT 0,
    amount_saved_from_cost_savings NUMERIC(20,2) NOT NULL DEFAULT 0,
    from_purchase_request          BOOLEAN       NOT NULL DEFAULT FALSE,
    branch_source_id               INTEGER,
    is_goods_orders                BOOLEAN       DEFAULT FALSE,
    product_type                   TEXT,
    pr_confirm_date                TIMESTAMP,
    origin                         TEXT,
    source_updated_at              TIMESTAMP,
    synced_at                      TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_po_company      ON procurement.purchase_orders (company_source_id);
CREATE INDEX IF NOT EXISTS idx_po_state        ON procurement.purchase_orders (state);
CREATE INDEX IF NOT EXISTS idx_po_date         ON procurement.purchase_orders (date_order);
CREATE INDEX IF NOT EXISTS idx_po_approve      ON procurement.purchase_orders (date_approve);
CREATE INDEX IF NOT EXISTS idx_po_date_planned ON procurement.purchase_orders (date_planned);
CREATE INDEX IF NOT EXISTS idx_po_partner      ON procurement.purchase_orders (partner_source_id);
CREATE INDEX IF NOT EXISTS idx_po_origin       ON procurement.purchase_orders (origin);

-- Goods Receipts / Receiving Notes
-- Sumber: public.stock_picking WHERE picking_type_code = 'incoming' (Hashmicro)
-- date_done = Date of Transfer / Received On.
CREATE TABLE IF NOT EXISTS procurement.goods_receipts (
    source_id           INTEGER   PRIMARY KEY,
    company_source_id   INTEGER   NOT NULL,
    po_source_id        INTEGER,
    partner_source_id   INTEGER,
    name                TEXT      NOT NULL,
    scheduled_date      TIMESTAMP,
    date_done           TIMESTAMP,
    is_on_time          BOOLEAN GENERATED ALWAYS AS (date_done <= scheduled_date) STORED,
    state               TEXT      NOT NULL,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gr_company   ON procurement.goods_receipts (company_source_id);
CREATE INDEX IF NOT EXISTS idx_gr_po        ON procurement.goods_receipts (po_source_id);
CREATE INDEX IF NOT EXISTS idx_gr_scheduled ON procurement.goods_receipts (scheduled_date);
CREATE INDEX IF NOT EXISTS idx_gr_date_done ON procurement.goods_receipts (date_done);
CREATE INDEX IF NOT EXISTS idx_gr_on_time   ON procurement.goods_receipts (is_on_time);

-- Purchase Order <-> Purchase Request relationship
-- Sumber: public.purchase_order_purchase_request_rel (Hashmicro)
CREATE TABLE IF NOT EXISTS procurement.purchase_order_purchase_requests (
    purchase_order_source_id   INTEGER   NOT NULL,
    purchase_request_source_id INTEGER   NOT NULL,
    synced_at                  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (purchase_order_source_id, purchase_request_source_id)
);

CREATE INDEX IF NOT EXISTS idx_po_pr_po ON procurement.purchase_order_purchase_requests (purchase_order_source_id);
CREATE INDEX IF NOT EXISTS idx_po_pr_pr ON procurement.purchase_order_purchase_requests (purchase_request_source_id);
