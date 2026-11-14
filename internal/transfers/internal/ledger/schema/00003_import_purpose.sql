-- +goose Up
-- S33 (Import pipeline): a Grab's Transfer becomes its Import's. The
-- carried Import takes its Grab's id, so job stays, and its one sent file
-- is the Import's first.
UPDATE transfers_file SET group_key = '0' WHERE transfer_id IN (SELECT id FROM transfers_record WHERE purpose = 'grab');
UPDATE transfers_record SET purpose = 'import' WHERE purpose = 'grab';
