package mock

import "github.com/slipstream/slipstream/internal/indexer/types"

// mockMovieCatalog contains all movies that should have mock releases.
// These align with the mock TMDB metadata.
var mockMovieCatalog = []MockMedia{
	{Title: "The Matrix", Year: 1999, TmdbID: 603, ImdbID: 133093},
	{Title: "Fight Club", Year: 1999, TmdbID: 550, ImdbID: 137523},
	{Title: "Pulp Fiction", Year: 1994, TmdbID: 680, ImdbID: 110912},
	{Title: "The Dark Knight", Year: 2008, TmdbID: 155, ImdbID: 468569},
	{Title: "The Shawshank Redemption", Year: 1994, TmdbID: 278, ImdbID: 111161},
	{Title: "The Godfather", Year: 1972, TmdbID: 238, ImdbID: 68646},
	{Title: "Inception", Year: 2010, TmdbID: 27205, ImdbID: 1375666},
	{Title: "Interstellar", Year: 2014, TmdbID: 157336, ImdbID: 816692},
	{Title: "The Lord of the Rings The Fellowship of the Ring", Year: 2001, TmdbID: 120, ImdbID: 120737},
	{Title: "The Avengers", Year: 2012, TmdbID: 24428, ImdbID: 848228},
	{Title: "Avengers Infinity War", Year: 2018, TmdbID: 299536, ImdbID: 4154756},
	{Title: "Avengers Endgame", Year: 2019, TmdbID: 299534, ImdbID: 4154796},
	{Title: "Spider-Man Across the Spider-Verse", Year: 2023, TmdbID: 569094, ImdbID: 9362722},
	{Title: "Dune", Year: 2021, TmdbID: 438631, ImdbID: 1160419},
	{Title: "Dune Part Two", Year: 2024, TmdbID: 693134, ImdbID: 15239678},
	{Title: "Ford v Ferrari", Year: 2019, TmdbID: 359724, ImdbID: 1950186},
	{Title: "Barbie", Year: 2023, TmdbID: 346698, ImdbID: 1517268},
	{Title: "Oppenheimer", Year: 2023, TmdbID: 872585, ImdbID: 15398776},
	{Title: "Avatar The Way of Water", Year: 2022, TmdbID: 76600, ImdbID: 1630029},
	{Title: "Avatar", Year: 2009, TmdbID: 19995, ImdbID: 499549},
	{Title: "Deadpool and Wolverine", Year: 2024, TmdbID: 533535, ImdbID: 6263850},
	{Title: "Everything Everywhere All at Once", Year: 2022, TmdbID: 545611, ImdbID: 6710474},
	{Title: "Guardians of the Galaxy Vol 3", Year: 2023, TmdbID: 447365, ImdbID: 6791350},
	{Title: "Venom The Last Dance", Year: 2024, TmdbID: 912649, ImdbID: 16366836},
	{Title: "Inside Out 2", Year: 2024, TmdbID: 1022789, ImdbID: 22022452},
}

// mockTVCatalog contains all TV series that should have mock releases.
// These align with the mock TMDB/TVDB metadata.
var mockTVCatalog = []MockMedia{
	{Title: "Game of Thrones", Year: 2011, TvdbID: 121361, ImdbID: 944947},
	{Title: "Breaking Bad", Year: 2008, TvdbID: 81189, ImdbID: 903747},
	{Title: "Stranger Things", Year: 2016, TvdbID: 305288, ImdbID: 4574334},
	{Title: "House of the Dragon", Year: 2022, TvdbID: 371572, ImdbID: 11198330},
	{Title: "The Flash", Year: 2014, TvdbID: 279121, ImdbID: 3107288},
	{Title: "Loki", Year: 2021, TvdbID: 362472, ImdbID: 9140554},
	{Title: "The Big Bang Theory", Year: 2007, TvdbID: 80379, ImdbID: 898266},
	{Title: "How I Met Your Mother", Year: 2005, TvdbID: 75760, ImdbID: 460649},
	{Title: "The Simpsons", Year: 1989, TvdbID: 71663, ImdbID: 96697},
	{Title: "South Park", Year: 1997, TvdbID: 75897, ImdbID: 121955},
	{Title: "Friends", Year: 1994, TvdbID: 79168, ImdbID: 108778},
	{Title: "The Walking Dead", Year: 2010, TvdbID: 153021, ImdbID: 1520211},
	{Title: "Money Heist", Year: 2017, TvdbID: 327417, ImdbID: 6468322},
	{Title: "The Boys", Year: 2019, TvdbID: 355567, ImdbID: 1190634},
	{Title: "Squid Game", Year: 2021, TvdbID: 383275, ImdbID: 10919420},
	{Title: "The Last of Us", Year: 2023, TvdbID: 392256, ImdbID: 3581920},
	{Title: "Better Call Saul", Year: 2015, TvdbID: 273181, ImdbID: 3032476},
	{Title: "Lucifer", Year: 2016, TvdbID: 295685, ImdbID: 4052886},
	{Title: "The Mandalorian", Year: 2019, TvdbID: 361753, ImdbID: 8111088},
	{Title: "Invincible", Year: 2021, TvdbID: 368207, ImdbID: 6741278},
	{Title: "Yellowstone", Year: 2018, TvdbID: 341164, ImdbID: 4236770},
	{Title: "WandaVision", Year: 2021, TvdbID: 362392, ImdbID: 9140560},
	{Title: "Ahsoka", Year: 2023, TvdbID: 393187, ImdbID: 13622776},
	{Title: "Arcane", Year: 2021, TvdbID: 371028, ImdbID: 11126994},
}

// buildMovieReleasesMap generates the movie releases map from the catalog.
func buildMovieReleasesMap() map[int][]types.ReleaseInfo {
	result := make(map[int][]types.ReleaseInfo)
	for _, m := range mockMovieCatalog {
		result[m.TmdbID] = generateMovieReleases(m)
	}
	return result
}

// buildTVReleasesMap generates the TV releases map from the catalog.
func buildTVReleasesMap() map[int][]types.ReleaseInfo {
	result := make(map[int][]types.ReleaseInfo)
	for _, m := range mockTVCatalog {
		result[m.TvdbID] = generateTVReleases(m)
	}
	return result
}
