-- +goose Up
-- What S35 (Media Servers) needs of a doomed file. origin is where its
-- bytes sat when the move it was doomed in was queued: the path a Media
-- Server lists it at while the move is pending. replaced_by is the file an
-- upgrade put in its cells, whose added pairs with this file's removed.
-- Files doomed before this migration have neither.
ALTER TABLE files_doomed ADD COLUMN origin TEXT CHECK (origin <> '');
ALTER TABLE files_doomed ADD COLUMN replaced_by BLOB;
