-- +goose Up
-- +goose StatementBegin

-- Slots 1 and 2 should always be enabled (no toggle in UI)
UPDATE version_slots SET enabled = 1 WHERE slot_number <= 2;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert to original disabled state
UPDATE version_slots SET enabled = 0 WHERE slot_number <= 2;

-- +goose StatementEnd
