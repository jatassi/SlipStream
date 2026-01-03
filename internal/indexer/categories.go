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

// CategoryName returns a human-readable name for a category.
func CategoryName(id int) string {
	names := map[int]string{
		CategoryConsole:       "Console",
		CategoryMovies:        "Movies",
		CategoryMoviesForeign: "Movies/Foreign",
		CategoryMoviesOther:   "Movies/Other",
		CategoryMoviesSD:      "Movies/SD",
		CategoryMoviesHD:      "Movies/HD",
		CategoryMoviesUHD:     "Movies/UHD",
		CategoryMoviesBluRay:  "Movies/BluRay",
		CategoryMovies3D:      "Movies/3D",
		CategoryMoviesDVD:     "Movies/DVD",
		CategoryMoviesWebDL:   "Movies/WEB-DL",
		CategoryAudio:         "Audio",
		CategoryPC:            "PC",
		CategoryTV:            "TV",
		CategoryTVForeign:     "TV/Foreign",
		CategoryTVOther:       "TV/Other",
		CategoryTVSD:          "TV/SD",
		CategoryTVHD:          "TV/HD",
		CategoryTVUHD:         "TV/UHD",
		CategoryTVSport:       "TV/Sport",
		CategoryTVAnime:       "TV/Anime",
		CategoryTVDoc:         "TV/Documentary",
		CategoryTVWebDL:       "TV/WEB-DL",
		CategoryXXX:           "XXX",
		CategoryBooks:         "Books",
		CategoryOther:         "Other",
	}
	if name, ok := names[id]; ok {
		return name
	}
	return "Unknown"
}

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

// IsMovieCategory returns true if the category is a movie category.
func IsMovieCategory(id int) bool {
	return id >= 2000 && id < 3000
}

// IsTVCategory returns true if the category is a TV category.
func IsTVCategory(id int) bool {
	return id >= 5000 && id < 6000
}

// DefaultMovieCategories returns the default categories to search for movies.
func DefaultMovieCategories() []int {
	return []int{
		CategoryMovies,
		CategoryMoviesSD,
		CategoryMoviesHD,
		CategoryMoviesUHD,
		CategoryMoviesBluRay,
		CategoryMoviesWebDL,
	}
}

// DefaultTVCategories returns the default categories to search for TV shows.
func DefaultTVCategories() []int {
	return []int{
		CategoryTV,
		CategoryTVSD,
		CategoryTVHD,
		CategoryTVUHD,
		CategoryTVAnime,
		CategoryTVWebDL,
	}
}

// ParseCategories converts category IDs to CategoryMapping objects.
func ParseCategories(ids []int) []CategoryMapping {
	mappings := make([]CategoryMapping, 0, len(ids))
	for _, id := range ids {
		mappings = append(mappings, CategoryMapping{
			ID:   id,
			Name: CategoryName(id),
		})
	}
	return mappings
}
