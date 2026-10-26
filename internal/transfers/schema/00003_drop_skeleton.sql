-- +goose Up
-- The walking skeleton's Grab-keyed record of one Transfer and its
-- verified chunks. The Controller records Transfers in library.db now, and
-- a receiver keeps its verified chunks in transfers_resume.
DROP TABLE transfers_chunk;
DROP TABLE transfers_transfer;
