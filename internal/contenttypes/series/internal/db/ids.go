package db

type (
	TVDBID int64
	TMDBID int64
	// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
	Singleton int64
)
