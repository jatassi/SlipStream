package preferences

// SeriesSearchOnAdd defines options for searching when adding a series
type SeriesSearchOnAdd string

const (
	SeriesSearchOnAddNo           SeriesSearchOnAdd = "no"
	SeriesSearchOnAddFirstEpisode SeriesSearchOnAdd = "first_episode"
	SeriesSearchOnAddFirstSeason  SeriesSearchOnAdd = "first_season"
	SeriesSearchOnAddLatestSeason SeriesSearchOnAdd = "latest_season"
	SeriesSearchOnAddAll          SeriesSearchOnAdd = "all"
)

// SeriesMonitorOnAdd defines options for monitoring when adding a series
type SeriesMonitorOnAdd string

const (
	SeriesMonitorOnAddNone         SeriesMonitorOnAdd = "none"
	SeriesMonitorOnAddFirstSeason  SeriesMonitorOnAdd = "first_season"
	SeriesMonitorOnAddLatestSeason SeriesMonitorOnAdd = "latest_season"
	SeriesMonitorOnAddFuture       SeriesMonitorOnAdd = "future"
	SeriesMonitorOnAddAll          SeriesMonitorOnAdd = "all"
)

// Setting keys for add-flow preferences
const (
	KeyMovieSearchOnAdd      = "addflow_movie_search_on_add"
	KeySeriesSearchOnAdd     = "addflow_series_search_on_add"
	KeySeriesMonitorOnAdd    = "addflow_series_monitor_on_add"
	KeySeriesIncludeSpecials = "addflow_series_include_specials"
)

// AddFlowPreferences contains all add-flow related preferences
type AddFlowPreferences struct {
	MovieSearchOnAdd      bool               `json:"movieSearchOnAdd"`
	SeriesSearchOnAdd     SeriesSearchOnAdd  `json:"seriesSearchOnAdd"`
	SeriesMonitorOnAdd    SeriesMonitorOnAdd `json:"seriesMonitorOnAdd"`
	SeriesIncludeSpecials bool               `json:"seriesIncludeSpecials"`
}

// DefaultPreferences returns the default values for add-flow preferences
func DefaultPreferences() AddFlowPreferences {
	return AddFlowPreferences{
		MovieSearchOnAdd:      false,
		SeriesSearchOnAdd:     SeriesSearchOnAddNo,
		SeriesMonitorOnAdd:    SeriesMonitorOnAddFuture,
		SeriesIncludeSpecials: false,
	}
}

// ValidSeriesSearchOnAdd checks if a value is a valid SeriesSearchOnAdd option
func ValidSeriesSearchOnAdd(s string) bool {
	switch SeriesSearchOnAdd(s) {
	case SeriesSearchOnAddNo, SeriesSearchOnAddFirstEpisode, SeriesSearchOnAddFirstSeason, SeriesSearchOnAddLatestSeason, SeriesSearchOnAddAll:
		return true
	}
	return false
}

// ValidSeriesMonitorOnAdd checks if a value is a valid SeriesMonitorOnAdd option
func ValidSeriesMonitorOnAdd(s string) bool {
	switch SeriesMonitorOnAdd(s) {
	case SeriesMonitorOnAddNone, SeriesMonitorOnAddFirstSeason, SeriesMonitorOnAddLatestSeason, SeriesMonitorOnAddFuture, SeriesMonitorOnAddAll:
		return true
	}
	return false
}
