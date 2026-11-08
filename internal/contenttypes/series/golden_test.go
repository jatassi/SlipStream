package series_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/release/golden"
)

var names = new(series.Series)

func title(name string) release.Read { return release.Title(name, names, release.BuiltIn()) }

func file(path string) release.Read { return release.File(path, names, release.BuiltIn()) }

// The series' Mark keys, which its Covers reads back: s1e2, s1e2-4, s1,
// s1-3, s1p2, a12, a12-14, a12.5, d2024-03-15 and d2024-03-15.2.
func episode(season, ep int) string { return fmt.Sprintf("s%de%d", season, ep) }

func episodes(season int, eps []int) string {
	return fmt.Sprintf("s%de%d-%d", season, eps[0], eps[len(eps)-1])
}

func absolute(n float64) string { return "a" + strconv.FormatFloat(n, 'f', -1, 64) }

func aired(day time.Time) string { return "d" + day.Format(time.DateOnly) }

func airedOn(r golden.Row) string {
	return aired(time.Date(r.Int("year"), time.Month(r.Int("month")), r.Int("day"), 0, 0, 0, 0, time.UTC))
}

// numbered is nil when n reads as Sonarr's series title with the marks and
// the span given, the marks in any order.
func numbered(n release.Name, seriesTitle string, span release.Span, keys ...string) error {
	return errors.Join(golden.Same("title", golden.FirstTitle(n), seriesTitle), marked(n, span, keys...))
}

func marked(n release.Name, span release.Span, keys ...string) error {
	var got []string
	for _, m := range n.Marks {
		got = append(got, m.Key)
	}
	slices.Sort(got)
	slices.Sort(keys)
	return errors.Join(golden.Same("marks", got, keys), golden.Same("span", n.Span, span))
}

func unreadTitle(param string) golden.Check {
	return func(r golden.Row) error { return golden.Unread(title(r.String(param)).Name) }
}

// group is nil when n reads want as its group, or no group when want is
// empty, as upstream's null.
func group(n release.Read, want string) error {
	if want == "" {
		return golden.Same("group", n.Facts.Group, release.Fact[string]{})
	}
	return golden.Same("group", n.Facts.Group, release.Known(want))
}

func languages(check func(release.Fact[[]release.Language]) error) golden.Check {
	return func(r golden.Row) error {
		return golden.LanguageCheck(r, title(r.String("postTitle")).Facts.Languages, check)
	}
}

func has(langs ...string) golden.Check {
	return languages(func(got release.Fact[[]release.Language]) error { return golden.HasLanguages(got, langs...) })
}

func exactly(langs ...string) golden.Check {
	return languages(func(got release.Fact[[]release.Language]) error { return golden.Languages(got, langs...) })
}

func season(r golden.Row) error {
	return numbered(title(r.String("postTitle")).Name, r.String("title"), release.Level, fmt.Sprintf("s%d", r.Int("season")))
}

func single(r golden.Row) error {
	return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, episode(r.Int("seasonNumber"), r.Int("episodeNumber")))
}

func daily(r golden.Row) error {
	return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, airedOn(r))
}

func airsIn(days int) string {
	return time.Now().AddDate(0, 0, days).Format("A.Talk.Show.2006.01.02.1080p.WEB-DL.H264-Sonarr")
}

func expand(layout string, day time.Time) string {
	return strings.NewReplacer("{year}", day.Format("2006"), "{month}", day.Format("01"), "{day}", day.Format("02")).Replace(layout)
}

func TestGoldenSingleEpisode(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/SingleEpisodeParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_single_episode":               single,
			"should_parse_japanese_variety_show_format": single,
			"should_parse_decimal_number_as_special":    single,
			"should_parse_split_episode":                single,
		},
	})
}

func TestGoldenMultiEpisode(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/MultiEpisodeParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_multiple_episodes": func(r golden.Row) error {
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.Several, episodes(r.Int("season"), r.Ints("episodes")))
			},
		},
	})
}

func TestGoldenAbsoluteEpisodeNumber(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/AbsoluteEpisodeNumberParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_absolute_numbers": func(r golden.Row) error {
				keys := []string{absolute(float64(r.Int("absoluteEpisodeNumber")))}
				if ep := r.Int("episodeNumber"); ep != 0 {
					keys = append(keys, episode(r.Int("seasonNumber"), ep))
				}
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, keys...)
			},
			"should_parse_absolute_specials": func(r golden.Row) error {
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, absolute(float64(r.Int("absoluteEpisodeNumber"))))
			},
			"should_parse_absolute_specials_without_absolute_number": func(r golden.Row) error {
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.SpanUnknown)
			},
			"should_parse_multi_episode_absolute_numbers": func(r golden.Row) error {
				key := fmt.Sprintf("a%d-%d", r.Int("firstAbsoluteEpisodeNumber"), r.Int("lastAbsoluteEpisodeNumber"))
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.Several, key)
			},
			"should_parse_anime_season_packs": func(r golden.Row) error {
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.Level, fmt.Sprintf("s%d", r.Int("seasonNumber")))
			},
			"should_handle_anime_recap_numbering": func(r golden.Row) error {
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, absolute(r.Float("specialEpisodeNumber")))
			},
			"should_handle_ambiguously_named_anime_files_in_batch_release": func(r golden.Row) error {
				path := `C:\Test\` + r.String("releaseName") + `\` + r.String("filename")
				return numbered(file(path).Name, r.String("title"), release.One, absolute(float64(r.Int("absoluteEpisodeNumber"))))
			},
			"should_parse_absolute_followed_by_standard_as_standard": func(r golden.Row) error {
				return numbered(title(r.String("releaseName")).Name, r.String("title"), release.One,
					episode(r.Int("seasonNumber"), r.Int("episodeNumber")), absolute(r.Float("specialEpisodeNumber")))
			},
		},
	})
}

func TestGoldenSeason(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/SeasonParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_full_season_release": season,
			"should_parse_season_extras":       season,
			"should_parse_season_subpack":      season,
			"should_parse_partial_season_release": func(r golden.Row) error {
				key := fmt.Sprintf("s%dp%d", r.Int("season"), r.Int("seasonPart"))
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.Several, key)
			},
			"should_parse_multi_season_release": func(r golden.Row) error {
				seasons := r.Ints("expectedSeasons")
				key := fmt.Sprintf("s%d-%d", seasons[0], seasons[len(seasons)-1])
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.Levels, key)
			},
			"should_not_be_multi_season_for_a_single_season_release": func(r golden.Row) error {
				return marked(title(r.String("postTitle")).Name, release.Level, fmt.Sprintf("s%d", r.Int("season")))
			},
		},
	})
}

func TestGoldenDailyEpisode(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/DailyEpisodeParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_daily_episode":                          daily,
			"should_parse_daily_episode_using_short_month_format": daily,
			"should_parse_daily_episode_with_multiple_parts": func(r golden.Row) error {
				key := fmt.Sprintf("%s.%d", airedOn(r), r.Int("part"))
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, key)
			},
			"should_not_accept_ancient_daily_series": func(r golden.Row) error {
				return golden.Unread(title(expand(r.String("title"), time.Date(1950, 10, 14, 0, 0, 0, 0, time.UTC))).Name)
			},
			"should_accept_dates_within_future_limit": func(r golden.Row) error {
				day := time.Now().AddDate(0, 0, 2)
				return marked(title(expand(r.String("title"), day)).Name, release.One, aired(day))
			},
			"should_parse_episodes_released_before_their_air_date": func(r golden.Row) error {
				days := r.Int("days")
				return marked(title(airsIn(days)).Name, release.One, aired(time.Now().AddDate(0, 0, days)))
			},
			"should_not_parse_dates_beyond_future_limit": func(r golden.Row) error {
				return golden.Unread(title(airsIn(r.Int("days"))).Name)
			},
			"should_not_parse_ambiguous_daily_episode": unreadTitle("postTitle"),
		},
	})
}

func TestGoldenPath(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/PathParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_from_path": func(r golden.Row) error {
				return marked(file(r.String("path")).Name, release.One, episode(r.Int("season"), r.Int("episode")))
			},
			"should_parse_multi_episode_from_path": func(r golden.Row) error {
				return numbered(file(r.String("path")).Name, r.String("title"), release.Several, episodes(r.Int("season"), r.Ints("episodes")))
			},
		},
	})
}

func TestGoldenHashedRelease(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/HashedReleaseFixture.json",
		Checks: map[string]golden.Check{
			"should_properly_parse_hashed_releases": func(r golden.Row) error {
				read := file(r.String("path"))
				_, quality, _ := strings.Cut(r.Ref("quality"), ".")
				return errors.Join(
					golden.Same("title", golden.FirstTitle(read.Name), r.String("title")),
					golden.SonarrQuality(quality).Read(read.Facts),
					group(read, r.String("releaseGroup")))
			},
		},
	})
}

func TestGoldenAnimeMetadata(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/AnimeMetadataParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_releasegroup_and_hash": func(r golden.Row) error {
				return group(title(r.String("postTitle")), r.String("subGroup"))
			},
		},
		Unported: map[string]string{
			"should_parse_release_hashes_with_10b_or_8b": "SlipStream keeps no release hash.",
		},
	})
}

func subgroup(r golden.Row) string { return r.String("subgroup") }

func anime(groupOf func(golden.Row) string) golden.Check {
	return func(r golden.Row) error {
		read := title(r.String("postTitle"))
		return errors.Join(group(read, groupOf(r)),
			numbered(read.Name, r.String("title"), release.One, absolute(float64(r.Int("absoluteEpisodeNumber")))))
	}
}

func TestGoldenUnicodeRelease(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/UnicodeReleaseParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_chinese_anime_releases":                anime(subgroup),
			"should_parse_chinese_anime_season_episode_releases": anime(subgroup),
			"should_parse_unbracketed_chinese_anime_releases":    anime(subgroup),
			"should_parse_false_positive_chinese_anime_releases": anime(subgroup),
			"should_parse_gm_team_releases_and_files":            anime(func(golden.Row) string { return "GM-Team" }),
			"should_parse_chinese_multiepisode_releases": func(r golden.Row) error {
				read := title(r.String("postTitle"))
				eps := r.Ints("absoluteEpisodeNumbers")
				key := fmt.Sprintf("a%d-%d", eps[0], eps[len(eps)-1])
				return errors.Join(group(read, subgroup(r)), numbered(read.Name, r.String("title"), release.Several, key))
			},
			"should_parse_unicode_digits": func(r golden.Row) error {
				return numbered(title(r.String("postTitle")).Name, r.String("title"), release.One, absolute(float64(r.Int("absoluteEpisodeNumber"))))
			},
			"should_parse_full_season_releases": func(r golden.Row) error {
				read := title(r.String("postTitle"))
				return errors.Join(group(read, r.String("releaseGroup")),
					numbered(read.Name, r.String("title"), release.Level, fmt.Sprintf("s%d", r.Int("season"))))
			},
			"should_parse_multi_episode_release": func(r golden.Row) error {
				read := title(r.String("postTitle"))
				return errors.Join(group(read, r.String("releaseGroup")),
					numbered(read.Name, r.String("title"), release.Several, episodes(r.Int("season"), r.Ints("episodes"))))
			},
		},
	})
}

func TestGoldenParser(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/ParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_series_name": func(r golden.Row) error {
				n := title(r.String("postTitle")).Name
				got := golden.FirstTitle(n)
				if golden.Unread(n) == nil {
					got = r.String("postTitle")
				}
				return golden.Same("clean title", names.CleanTitle(got), names.CleanTitle(r.String("title")))
			},
			"should_parse_series_title_info": func(r golden.Row) error {
				want := release.Fact[int]{}
				if y := r.Int("year"); y != 0 {
					want = release.Known(y)
				}
				return golden.Same("year", title(r.String("postTitle")).Name.Year, want)
			},
			"should_clean_up_invalid_path_characters": func(r golden.Row) error {
				title(r.String("postTitle"))
				return nil
			},
			"should_remove_request_info_from_title": func(r golden.Row) error {
				return golden.Same("title", golden.FirstTitle(title(r.String("postTitle")).Name), r.String("title"))
			},
			"should_parse_quality_from_extension": func(r golden.Row) error {
				_, ok := title(r.String("title")).Facts.Quality.Get()
				return golden.Same("quality known", ok, true)
			},
			"should_parse_multiple_series_titles": func(r golden.Row) error {
				got := slices.Sorted(slices.Values(title(r.String("postTitle")).Name.Titles))
				return golden.Same("titles", got, slices.Sorted(slices.Values(r.Strings("titles"))))
			},
			"should_parse_language_after_parsing_title": func(r golden.Row) error {
				return golden.Languages(title(r.String("postTitle")).Facts.Languages, r.String("expectedLanguage"))
			},
		},
		Unported: map[string]string{
			"should_parse_releasetitle": "SlipStream keeps the name as given and reads no release title from it.",
		},
	})
}

func TestGoldenCrap(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/CrapParserFixture.json",
		Checks: map[string]golden.Check{
			"should_not_parse_crap":                             unreadTitle("title"),
			"should_not_parse_file_name_without_proper_spacing": unreadTitle("fileName"),
			"should_not_parse_invalid_release_name":             unreadTitle("fileName"),
			"should_not_parse_special_with_part_number":         unreadTitle("fileName"),
			"should_parse_unknown_formats_without_error": func(r golden.Row) error {
				if golden.Unread(title(r.String("title")).Name) == nil {
					return errors.New("nothing reads from the name, want a title or a mark")
				}
				return nil
			},
			"should_not_parse_random": func(r golden.Row) error {
				var errs []error
				length := r.Int("length")
				rng := rand.New(rand.NewPCG(uint64(length), 0)) //nolint:gosec // a fixed draw the test replays
				for _, name := range golden.RandomNames(rng, length) {
					if err := golden.Unread(title(name).Name); err != nil {
						errs = append(errs, fmt.Errorf("%s: %w", name, err))
					}
				}
				return errors.Join(errs...)
			},
		},
	})
}

func TestGoldenNormalizeTitle(t *testing.T) {
	checks := golden.Cleaning(names.CleanTitle)
	checks["should_replace_percent_sign_with_percent_following_numbers"] = func(r golden.Row) error {
		return golden.Same("clean title", names.CleanTitle(r.String("input")), r.String("expected"))
	}
	golden.Run(t, golden.Fixture{Path: "testdata/upstream/sonarr/NormalizeSeriesTitleFixture.json", Checks: checks})
}

func TestGoldenLanguage(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/LanguageParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_language_unknown":                                                 has("Unknown"),
			"should_parse_language_english":                                                 has("English"),
			"should_parse_language_french":                                                  has("French"),
			"should_parse_language_french_english":                                          has("French", "English"),
			"should_parse_language_spanish":                                                 has("Spanish"),
			"should_parse_language_german":                                                  has("German"),
			"should_parse_language_italian":                                                 has("Italian"),
			"should_parse_language_danish":                                                  has("Danish"),
			"should_parse_language_dutch":                                                   has("Dutch"),
			"should_parse_language_japanese":                                                has("Japanese"),
			"should_parse_language_icelandic":                                               has("Icelandic"),
			"should_parse_language_chinese":                                                 has("Chinese"),
			"should_parse_language_korean":                                                  has("Korean"),
			"should_parse_language_latvian":                                                 has("Latvian"),
			"should_parse_language_russian":                                                 has("Russian"),
			"should_parse_language_polish":                                                  has("Polish"),
			"should_parse_language_vietnamese":                                              has("Vietnamese"),
			"should_parse_language_swedish":                                                 has("Swedish"),
			"should_parse_language_norwegian":                                               has("Norwegian"),
			"should_parse_language_finnish":                                                 has("Finnish"),
			"should_parse_language_turkish":                                                 has("Turkish"),
			"should_parse_language_portuguese":                                              has("Portuguese"),
			"should_parse_language_flemish":                                                 has("Flemish"),
			"should_parse_language_greek":                                                   has("Greek"),
			"should_parse_language_hungarian":                                               has("Hungarian"),
			"should_parse_language_hebrew":                                                  has("Hebrew"),
			"should_parse_language_lithuanian":                                              has("Lithuanian"),
			"should_parse_language_czech":                                                   has("Czech"),
			"should_parse_language_arabic":                                                  has("Arabic"),
			"should_parse_language_hindi":                                                   has("Hindi"),
			"should_parse_language_bulgarian":                                               has("Bulgarian"),
			"should_parse_language_malayalam":                                               has("Malayalam"),
			"should_parse_language_ukrainian":                                               has("Ukrainian"),
			"should_parse_language_slovak":                                                  has("Slovak"),
			"should_parse_language_thai":                                                    has("Thai"),
			"should_parse_language_portuguese_brazil":                                       has("PortugueseBrazil"),
			"should_parse_language_spanish_latino":                                          has("SpanishLatino"),
			"should_parse_language_romanian":                                                has("Romanian"),
			"should_parse_language_catalan":                                                 has("Catalan"),
			"should_parse_hindi_and_english":                                                exactly("Hindi", "English"),
			"should_parse_spanish_and_catalan":                                              exactly("Spanish", "Catalan"),
			"should_parse_english_spanish_and_catalan":                                      exactly("English", "Spanish", "Catalan"),
			"should_add_original_language_to_german_release_with_dl_tag":                    exactly("German", "Original"),
			"should_not_add_original_language_to_german_release_when_title_contains_web_dl": exactly("German"),
			"should_add_original_language_and_english_to_german_release_with_ml_tag":        exactly("German", "Original", "English"),
			"should_parse_original_title_from_release_name":                                 exactly("Original"),
			"should_parse_english_and_ukranian":                                             exactly("Ukrainian", "English"), //nolint:misspell // upstream's method name
			"should_parse_turkish_and_ukranian":                                             exactly("Ukrainian", "Turkish"), //nolint:misspell // upstream's method name
			"should_parse_azerbaijani":                                                      exactly("Azerbaijani"),
			"should_parse_uzbek":                                                            exactly("Uzbek"),
			"should_parse_language_urdu":                                                    has("Urdu"),
			"should_parse_language_romansh":                                                 has("Romansh"),
			"should_parse_language_georgian":                                                has("Georgian"),
			"should_parse_language_russian_and_georgian":                                    exactly("Russian", "Georgian"),
			"should_parse_language_irish":                                                   has("Irish"),
			"should_parse_language_english_french_german_italian_portuguese_spanish":        exactly("English", "French", "German", "Italian", "Portuguese", "Spanish"),
		},
		Unported: map[string]string{
			"should_parse_subtitle_language_unknown": subtitles,
			"should_parse_subtitle_language_english": subtitles,
			"should_parse_title_and_tags":            subtitles,
			"should_not_parse_false_title":           subtitles,
		},
	})
}

const subtitles = "SlipStream reads no subtitle file names."
