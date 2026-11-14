-- +goose Up
-- The Media Servers section's stored values, each NULL at its default. The
-- renewal cadence is the task's own row in tasks_task.
CREATE TABLE mediaservers_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    wait_seconds INTEGER CHECK (wait_seconds >= 0),
    sessions_every_seconds INTEGER CHECK (sessions_every_seconds > 0),
    unhealthy_after_seconds INTEGER CHECK (unhealthy_after_seconds > 0),
    keep_plays_seconds INTEGER CHECK (keep_plays_seconds > 0),
    renew_margin_seconds INTEGER CHECK (renew_margin_seconds > 0)
) STRICT;

INSERT INTO mediaservers_settings (id) VALUES (1);
