-- +goose Up
-- grabs_grab: found_by, what found its Release, written with the Grab so it
-- outlives the run that made it. A Grab whose making run is gone reads
-- search.
ALTER TABLE grabs_grab ADD COLUMN found_by TEXT NOT NULL DEFAULT 'search'
    CHECK (found_by IN ('search', 'backlog', 'rss', 'failed_grab', 'manual'));

UPDATE grabs_grab SET found_by = CASE
    WHEN source IN ('manual', 'rss') THEN source
    ELSE coalesce((
        SELECT s.trigger FROM grabs_search_grab sg JOIN grabs_search s ON s.id = sg.search_id
        WHERE sg.grab_id = grabs_grab.id AND sg.made = 1 AND s.trigger IN ('backlog', 'failed_grab')), 'search')
END;
