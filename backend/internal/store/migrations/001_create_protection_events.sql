CREATE TABLE protection_events (
    id             BIGSERIAL PRIMARY KEY,
    device_id      TEXT             NOT NULL,
    element        TEXT             NOT NULL,
    name           TEXT             NOT NULL,
    phase          TEXT             NOT NULL DEFAULT '',
    value          DOUBLE PRECISION NOT NULL,
    threshold      DOUBLE PRECISION NOT NULL,
    unit           TEXT             NOT NULL,
    picked_up_at   TIMESTAMPTZ      NOT NULL,
    dropped_out_at TIMESTAMPTZ,
    CONSTRAINT dropout_after_pickup
        CHECK (dropped_out_at IS NULL OR dropped_out_at >= picked_up_at)
);

-- Fast lookups for "events on RELAY-02, newest first"
CREATE INDEX idx_protection_events_device_time
    ON protection_events (device_id, picked_up_at DESC);

-- At most ONE active alarm per device + element + phase
CREATE UNIQUE INDEX uniq_active_condition
    ON protection_events (device_id, element, phase)
    WHERE dropped_out_at IS NULL;
