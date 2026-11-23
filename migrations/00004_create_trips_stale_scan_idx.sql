-- +goose NO TRANSACTION

-- +goose Up

CREATE INDEX CONCURRENTLY trips_active_stale_scan_idx
    ON trips ((COALESCE(last_position_at, started_at)), id)
    WHERE status = 'active';

-- +goose Down

DROP INDEX CONCURRENTLY trips_active_stale_scan_idx;
