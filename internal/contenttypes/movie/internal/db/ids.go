package db

type (
	TMDBID       int64
	IMDbID       string
	PersonID     int64
	CompanyID    int64
	CollectionID int64
	// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
	Singleton int64
	// CountsAs is which release makes a movie count as out.
	CountsAs string
)
