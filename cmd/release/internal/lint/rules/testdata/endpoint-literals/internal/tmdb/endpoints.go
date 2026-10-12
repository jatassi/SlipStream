package tmdb

type Endpoints struct {
	API string
}

func DefaultEndpoints() Endpoints {
	return Endpoints{API: "https://api.themoviedb.org/3"}
}

type Config struct {
	API string
}

func (Config) DefaultEndpoints() Endpoints {
	return Endpoints{API: "https://image.tmdb.org"}
}
