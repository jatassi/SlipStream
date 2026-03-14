package indexer

// Standard Newznab Categories
// https://newznab.readthedocs.io/en/latest/misc/api/#predefined-categories
const (
	// Main categories
	CategoryConsole = 1000
	CategoryMovies  = 2000
	CategoryAudio   = 3000
	CategoryPC      = 4000
	CategoryTV      = 5000
	CategoryXXX     = 6000
	CategoryBooks   = 7000
	CategoryOther   = 8000

	// Movies subcategories
	CategoryMoviesForeign = 2010
	CategoryMoviesOther   = 2020
	CategoryMoviesSD      = 2030
	CategoryMoviesHD      = 2040
	CategoryMoviesUHD     = 2045
	CategoryMoviesBluRay  = 2050
	CategoryMovies3D      = 2060
	CategoryMoviesDVD     = 2070
	CategoryMoviesWebDL   = 2080

	// TV subcategories
	CategoryTVForeign = 5010
	CategoryTVOther   = 5020
	CategoryTVSD      = 5030
	CategoryTVHD      = 5040
	CategoryTVUHD     = 5045
	CategoryTVSport   = 5060
	CategoryTVAnime   = 5070
	CategoryTVDoc     = 5080
	CategoryTVWebDL   = 5090
)

// MovieCategories returns all movie-related categories.
func MovieCategories() []int {
	return []int{
		CategoryMovies,
		CategoryMoviesForeign,
		CategoryMoviesOther,
		CategoryMoviesSD,
		CategoryMoviesHD,
		CategoryMoviesUHD,
		CategoryMoviesBluRay,
		CategoryMovies3D,
		CategoryMoviesDVD,
		CategoryMoviesWebDL,
	}
}

// TVCategories returns all TV-related categories.
func TVCategories() []int {
	return []int{
		CategoryTV,
		CategoryTVForeign,
		CategoryTVOther,
		CategoryTVSD,
		CategoryTVHD,
		CategoryTVUHD,
		CategoryTVSport,
		CategoryTVAnime,
		CategoryTVDoc,
		CategoryTVWebDL,
	}
}
