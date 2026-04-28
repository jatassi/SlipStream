-- +goose Up
-- Add the CAM quality tier (id 18) to every existing quality profile, defaulting
-- to allowed=false so that camcorder/TELESYNC releases are filtered out unless a
-- user explicitly opts in. Skips profiles that already include id 18.
UPDATE quality_profiles
SET items = json_insert(
        items,
        '$[#]',
        json('{"quality":{"id":18,"name":"CAM","source":"cam","resolution":0,"weight":0},"allowed":false}')
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1
    FROM json_each(quality_profiles.items)
    WHERE json_extract(json_each.value, '$.quality.id') = 18
);

-- +goose Down
-- Remove the CAM tier (id 18) from every profile.
UPDATE quality_profiles
SET items = (
        SELECT json_group_array(json(value))
        FROM json_each(quality_profiles.items)
        WHERE json_extract(value, '$.quality.id') != 18
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE EXISTS (
    SELECT 1
    FROM json_each(quality_profiles.items)
    WHERE json_extract(json_each.value, '$.quality.id') = 18
);
