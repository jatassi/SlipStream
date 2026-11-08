// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/RuTracker.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package rutracker

import (
	_ "embed"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[RuTrackerSettings]

var Port = port.Define(port.Entry{
	ID:   "rutracker",
	Name: "RuTracker.org",
	IndexerURLs: []string{
		"https://rutracker.org/",
		"https://rutracker.net/",
		"https://rutracker.nl/",
	},
	// SlipStream: SlipStream's own sentence, per WRITING.md.
	Description:  "RuTracker.org is a semi-private Russian Tracker for movies, TV, music, books, games and software.",
	Language:     "ru-RU",
	Encoding:     "windows-1251",
	Privacy:      port.SemiPrivate,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[RuTrackerSettings]{
	Login:    port.FormLogin[RuTrackerSettings]{DoLogin: DoLogin, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie:    GetSearchRequests,
	TV:       GetTvSearchRequests,
	Basic:    GetSearchRequests,
	Parse:    ParseResponse,
	Download: Download,
})

func Download(ix *ix, link *url.URL) (port.Got, error) {
	if ix.Settings.UseMagnetLinks && strings.Contains(link.RequestURI(), "viewtopic.php?t=") {
		response, err := ix.Fetch(port.Get(link).Accept("text/html"))
		if err != nil {
			return nil, err
		}

		magnetLink := dom.Parse(response.Content).QuerySelector(`table.attach a.magnet-link[href^="magnet:?"]`).GetAttribute("href")

		if magnetLink == "" {
			return nil, port.IndexerError("the topic page at %s has no Magnet link", link)
		}

		return port.Magnet{Link: magnetLink}, nil
	}

	return ix.Download(link)
}

func DoLogin(in *port.SignIn[RuTrackerSettings]) error {
	loginURL := in.BaseURL.JoinPath("forum/login.php")

	// SlipStream: a sign-in's requests carry only the cookies it earns, so
	// Cookies = null needs no line, and the framework keeps the session it
	// earns in place of UpdateCookies.
	response, err := in.Do(port.Post(loginURL).
		AddFormParameter("login_username", in.Settings.Username).
		AddFormParameter("login_password", string(in.Settings.Password)).
		AddFormParameter("login", "Login").
		AddFormParameter("redirect", "index.php").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Referer", loginURL.String()))
	if err != nil {
		return err
	}

	if !strings.Contains(response.Content, `id="logged-in-username"`) {
		document := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(document.QuerySelector("h4.warnColor1.tCenter.mrg_16, div.msg-main").TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func CheckIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return strings.Contains(httpResponse.RedirectURL, "login.php") || !strings.Contains(httpResponse.Content, `id="logged-in-username"`)
}

func SetCapabilities() port.Capabilities {
	// SlipStream: MusicSearchParams and BookSearchParams have no mode in
	// SlipStream, and SupportsRawSearch no meaning. Music and book
	// searches go through Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		// Note: When refreshing the categories use the tracker.php page and NOT the search.php page!
		Categories: port.Categories(
			port.Map(22, newznab.Movies, "Наше кино"),
			port.Map(941, newznab.Movies, "|- Кино СССР"),
			port.Map(1666, newznab.Movies, "|- Детские отечественные фильмы"),
			port.Map(376, newznab.Movies, "|- Авторские дебюты"),
			port.Map(106, newznab.Movies, "|- Фильмы России и СССР на национальных языках [без перевода]"),
			port.Map(7, newznab.MoviesForeign, "Зарубежное кино"),
			port.Map(187, newznab.MoviesForeign, "|- Классика мирового кинематографа"),
			port.Map(2090, newznab.MoviesForeign, "|- Фильмы до 1990 года"),
			port.Map(2221, newznab.MoviesForeign, "|- Фильмы 1991-2000"),
			port.Map(2091, newznab.MoviesForeign, "|- Фильмы 2001-2005"),
			port.Map(2092, newznab.MoviesForeign, "|- Фильмы 2006-2010"),
			port.Map(2093, newznab.MoviesForeign, "|- Фильмы 2011-2015"),
			port.Map(2200, newznab.MoviesForeign, "|- Фильмы 2016-2020"),
			port.Map(1950, newznab.MoviesForeign, "|- Фильмы 2021-2023"),
			port.Map(252, newznab.MoviesForeign, "|- Фильмы 2024"),
			port.Map(2540, newznab.MoviesForeign, "|- Фильмы Ближнего Зарубежья"),
			port.Map(934, newznab.MoviesForeign, "|- Азиатские фильмы"),
			port.Map(505, newznab.MoviesForeign, "|- Индийское кино"),
			port.Map(212, newznab.MoviesForeign, "|- Сборники фильмов"),
			port.Map(2459, newznab.MoviesForeign, "|- Короткий метр"),
			port.Map(1235, newznab.MoviesForeign, "|- Грайндхаус"),
			port.Map(166, newznab.MoviesForeign, "|- Зарубежные фильмы без перевода"),
			port.Map(185, newznab.Audio, "|- Звуковые дорожки и Переводы"),
			port.Map(124, newznab.MoviesOther, "Арт-хаус и авторское кино"),
			port.Map(1543, newznab.MoviesOther, "|- Короткий метр (Арт-хаус и авторское кино)"),
			port.Map(709, newznab.MoviesOther, "|- Документальные фильмы (Арт-хаус и авторское кино)"),
			port.Map(1577, newznab.MoviesOther, "|- Анимация (Арт-хаус и авторское кино)"),
			port.Map(511, newznab.TVOther, "Театр"),
			port.Map(1493, newznab.TVOther, "|- Спектакли без перевода"),
			port.Map(93, newznab.MoviesDVD, "DVD Video"),
			port.Map(905, newznab.MoviesDVD, "|- Классика мирового кинематографа (DVD Video)"),
			port.Map(101, newznab.MoviesDVD, "|- Зарубежное кино (DVD Video)"),
			port.Map(100, newznab.MoviesDVD, "|- Наше кино (DVD Video)"),
			port.Map(877, newznab.MoviesDVD, "|- Фильмы Ближнего Зарубежья (DVD Video)"),
			port.Map(1576, newznab.MoviesDVD, "|- Азиатские фильмы (DVD Video)"),
			port.Map(572, newznab.MoviesDVD, "|- Арт-хаус и авторское кино (DVD Video)"),
			port.Map(2220, newznab.MoviesDVD, "|- Индийское кино (DVD Video)"),
			port.Map(1670, newznab.MoviesDVD, "|- Грайндхаус (DVD Video)"),
			port.Map(2198, newznab.MoviesHD, "HD Video"),
			port.Map(2199, newznab.MoviesHD, "|- Классика мирового кинематографа (HD Video)"),
			port.Map(313, newznab.MoviesHD, "|- Зарубежное кино (HD Video)"),
			port.Map(312, newznab.MoviesHD, "|- Наше кино (HD Video)"),
			port.Map(1247, newznab.MoviesHD, "|- Фильмы Ближнего Зарубежья (HD Video)"),
			port.Map(2201, newznab.MoviesHD, "|- Азиатские фильмы (HD Video)"),
			port.Map(2339, newznab.MoviesHD, "|- Арт-хаус и авторское кино (HD Video)"),
			port.Map(140, newznab.MoviesHD, "|- Индийское кино (HD Video)"),
			port.Map(194, newznab.MoviesHD, "|- Грайндхаус (HD Video)"),
			port.Map(718, newznab.MoviesUHD, "UHD Video"),
			port.Map(775, newznab.MoviesUHD, "|- Классика мирового кинематографа (UHD Video)"),
			port.Map(1457, newznab.MoviesUHD, "|- Зарубежное кино (UHD Video)"),
			port.Map(1940, newznab.MoviesUHD, "|- Наше кино (UHD Video)"),
			port.Map(272, newznab.MoviesUHD, "|- Азиатские фильмы (UHD Video)"),
			port.Map(271, newznab.MoviesUHD, "|- Арт-хаус и авторское кино (UHD Video)"),
			port.Map(352, newznab.Movies3D, "3D/Стерео Кино, Видео, TV и Спорт"),
			port.Map(549, newznab.Movies3D, "|- 3D Кинофильмы"),
			port.Map(1213, newznab.Movies3D, "|- 3D Мультфильмы"),
			port.Map(2109, newznab.Movies3D, "|- 3D Документальные фильмы"),
			port.Map(514, newznab.Movies3D, "|- 3D Спорт"),
			port.Map(2097, newznab.Movies3D, "|- 3D Ролики, Музыкальное видео, Трейлеры к фильмам"),
			port.Map(4, newznab.Movies, "Мультфильмы"),
			port.Map(84, newznab.MoviesUHD, "|- Мультфильмы (UHD Video)"),
			port.Map(2343, newznab.MoviesHD, "|- Отечественные мультфильмы (HD Video)"),
			port.Map(930, newznab.MoviesHD, "|- Иностранные мультфильмы (HD Video)"),
			port.Map(2365, newznab.MoviesHD, "|- Иностранные короткометражные мультфильмы (HD Video)"),
			port.Map(1900, newznab.MoviesDVD, "|- Отечественные мультфильмы (DVD)"),
			port.Map(2258, newznab.MoviesDVD, "|- Иностранные короткометражные мультфильмы (DVD)"),
			port.Map(521, newznab.MoviesDVD, "|- Иностранные мультфильмы (DVD)"),
			port.Map(208, newznab.Movies, "|- Отечественные мультфильмы"),
			port.Map(539, newznab.Movies, "|- Отечественные полнометражные мультфильмы"),
			port.Map(2183, newznab.MoviesForeign, "|- Мультфильмы Ближнего Зарубежья"),
			port.Map(209, newznab.MoviesForeign, "|- Иностранные мультфильмы"),
			port.Map(484, newznab.MoviesForeign, "|- Иностранные короткометражные мультфильмы"),
			port.Map(822, newznab.Movies, "|- Сборники мультфильмов"),
			port.Map(181, newznab.Movies, "|- Мультфильмы без перевода"),
			port.Map(921, newznab.TV, "Мультсериалы"),
			port.Map(815, newznab.TVSD, "|- Мультсериалы (SD Video)"),
			port.Map(816, newznab.TVHD, "|- Мультсериалы (DVD Video)"),
			port.Map(1460, newznab.TVHD, "|- Мультсериалы (HD Video)"),
			port.Map(498, newznab.TVUHD, "|- Мультсериалы (UHD Video)"),
			port.Map(33, newznab.TVAnime, "Аниме"),
			port.Map(1106, newznab.TVAnime, "|- Онгоинги (HD Video)"),
			port.Map(1105, newznab.TVAnime, "|- Аниме (HD Video)"),
			port.Map(599, newznab.TVAnime, "|- Аниме (DVD)"),
			port.Map(1389, newznab.TVAnime, "|- Аниме (основной подраздел)"),
			port.Map(1391, newznab.TVAnime, "|- Аниме (плеерный подраздел)"),
			port.Map(2491, newznab.TVAnime, "|- Аниме (QC подраздел)"),
			port.Map(2544, newznab.TVAnime, "|- Ван-Пис"),
			port.Map(1642, newznab.TVAnime, "|- Гандам"),
			port.Map(1390, newznab.TVAnime, "|- Наруто"),
			port.Map(404, newznab.TVAnime, "|- Покемоны"),
			port.Map(893, newznab.TVAnime, "|- Японские мультфильмы"),
			port.Map(809, newznab.Audio, "|- Звуковые дорожки (Аниме)"),
			port.Map(2484, newznab.TVAnime, "|- Артбуки и журналы (Аниме)"),
			port.Map(1386, newznab.TVAnime, "|- Обои, сканы, аватары, арт"),
			port.Map(1387, newznab.TVAnime, "|- AMV и другие ролики"),
			port.Map(9, newznab.TV, "Русские сериалы"),
			port.Map(81, newznab.TVHD, "|- Русские сериалы (HD Video)"),
			port.Map(920, newznab.TVSD, "|- Русские сериалы (DVD Video)"),
			port.Map(80, newznab.TV, "|- Сельский детектив"),
			port.Map(1535, newznab.TV, "|- По законам военного времени"),
			port.Map(188, newznab.TV, "|- Московские тайны"),
			port.Map(91, newznab.TV, "|- Я знаю твои секреты"),
			port.Map(990, newznab.TV, "|- Универ / Универ. Новая общага / СашаТаня"),
			port.Map(1408, newznab.TV, "|- Женская версия"),
			port.Map(175, newznab.TV, "|- След"),
			port.Map(79, newznab.TV, "|- Некрасивая подружка"),
			port.Map(104, newznab.TV, "|- Психология преступления"),
			port.Map(189, newznab.TVForeign, "Зарубежные сериалы"),
			port.Map(842, newznab.TVForeign, "|- Новинки и сериалы в стадии показа"),
			port.Map(235, newznab.TVForeign, "|- Сериалы США и Канады"),
			port.Map(242, newznab.TVForeign, "|- Сериалы Великобритании и Ирландии"),
			port.Map(819, newznab.TVForeign, "|- Скандинавские сериалы"),
			port.Map(1531, newznab.TVForeign, "|- Испанские сериалы"),
			port.Map(721, newznab.TVForeign, "|- Итальянские сериалы"),
			port.Map(1102, newznab.TVForeign, "|- Европейские сериалы"),
			port.Map(1120, newznab.TVForeign, "|- Сериалы стран Африки, Ближнего и Среднего Востока"),
			port.Map(1214, newznab.TVForeign, "|- Сериалы Австралии и Новой Зеландии"),
			port.Map(489, newznab.TVForeign, "|- Сериалы Ближнего Зарубежья"),
			port.Map(387, newznab.TVForeign, "|- Сериалы совместного производства нескольких стран"),
			port.Map(1359, newznab.TVForeign, "|- Веб-сериалы, Вебизоды к сериалам и Пилотные серии сериалов"),
			port.Map(184, newznab.TVForeign, "|- Бесстыжие / Shameless (US)"),
			port.Map(1171, newznab.TVForeign, "|- Викинги / Vikings"),
			port.Map(1417, newznab.TVForeign, "|- Во все тяжкие / Breaking Bad"),
			port.Map(625, newznab.TVForeign, "|- Доктор Хаус / House M.D."),
			port.Map(1449, newznab.TVForeign, "|- Игра престолов / Game of Thrones"),
			port.Map(273, newznab.TVForeign, "|- Карточный Домик / House of Cards"),
			port.Map(504, newznab.TVForeign, "|- Клан Сопрано / The Sopranos"),
			port.Map(372, newznab.TVForeign, "|- Сверхъестественное / Supernatural"),
			port.Map(110, newznab.TVForeign, "|- Секретные материалы / The X-Files"),
			port.Map(121, newznab.TVForeign, "|- Твин пикс / Twin Peaks"),
			port.Map(507, newznab.TVForeign, "|- Теория большого взрыва + Детство Шелдона"),
			port.Map(536, newznab.TVForeign, "|- Форс-мажоры / Костюмы в законе / Suits"),
			port.Map(1144, newznab.TVForeign, "|- Ходячие мертвецы + Бойтесь ходячих мертвецов"),
			port.Map(173, newznab.TVForeign, "|- Черное зеркало / Black Mirror"),
			port.Map(195, newznab.TVForeign, "|- Для некондиционных раздач"),
			port.Map(2366, newznab.TVHD, "Зарубежные сериалы (HD Video)"),
			port.Map(119, newznab.TVUHD, "|- Зарубежные сериалы (UHD Video)"),
			port.Map(1803, newznab.TVHD, "|- Новинки и сериалы в стадии показа (HD Video)"),
			port.Map(266, newznab.TVHD, "|- Сериалы США и Канады (HD Video)"),
			port.Map(193, newznab.TVHD, "|- Сериалы Великобритании и Ирландии (HD Video)"),
			port.Map(1690, newznab.TVHD, "|- Скандинавские сериалы (HD Video)"),
			port.Map(1459, newznab.TVHD, "|- Европейские сериалы (HD Video)"),
			port.Map(1463, newznab.TVHD, "|- Сериалы стран Африки, Ближнего и Среднего Востока (HD Video)"),
			port.Map(825, newznab.TVHD, "|- Сериалы Австралии и Новой Зеландии (HD Video)"),
			port.Map(1248, newznab.TVHD, "|- Сериалы Ближнего Зарубежья (HD Video)"),
			port.Map(1288, newznab.TVHD, "|- Сериалы совместного производства нескольких стран (HD Video)"),
			port.Map(1669, newznab.TVHD, "|- Викинги / Vikings (HD Video)"),
			port.Map(2393, newznab.TVHD, "|- Доктор Хаус / House M.D. (HD Video)"),
			port.Map(265, newznab.TVHD, "|- Игра престолов / Game of Thrones (HD Video)"),
			port.Map(2406, newznab.TVHD, "|- Карточный домик (HD Video)"),
			port.Map(2404, newznab.TVHD, "|- Сверхъестественное / Supernatural (HD Video)"),
			port.Map(2405, newznab.TVHD, "|- Секретные материалы / The X-Files (HD Video)"),
			port.Map(2370, newznab.TVHD, "|- Твин пикс / Twin Peaks (HD Video)"),
			port.Map(2396, newznab.TVHD, "|- Теория Большого Взрыва / The Big Bang Theory (HD Video)"),
			port.Map(2398, newznab.TVHD, "|- Ходячие мертвецы + Бойтесь ходячих мертвецов (HD Video)"),
			port.Map(1949, newznab.TVHD, "|- Черное зеркало / Black Mirror (HD Video)"),
			port.Map(1498, newznab.TVHD, "|- Для некондиционных раздач (HD Video)"),
			port.Map(911, newznab.TVForeign, "Сериалы Латинской Америки, Турции и Индии"),
			port.Map(325, newznab.TVForeign, "|- Сериалы Аргентины"),
			port.Map(534, newznab.TVForeign, "|- Сериалы Бразилии"),
			port.Map(594, newznab.TVForeign, "|- Сериалы Венесуэлы"),
			port.Map(1301, newznab.TVForeign, "|- Сериалы Индии"),
			port.Map(607, newznab.TVForeign, "|- Сериалы Колумбии"),
			port.Map(1574, newznab.TVForeign, "|- Сериалы Латинской Америки с озвучкой (раздачи папками)"),
			port.Map(1539, newznab.TVForeign, "|- Сериалы Латинской Америки с субтитрами"),
			port.Map(694, newznab.TVForeign, "|- Сериалы Мексики"),
			port.Map(781, newznab.TVForeign, "|- Сериалы совместного производства"),
			port.Map(704, newznab.TVForeign, "|- Сериалы Турции"),
			port.Map(1537, newznab.TVForeign, "|- Для некондиционных раздач"),
			port.Map(2100, newznab.TVForeign, "Азиатские сериалы"),
			port.Map(820, newznab.TVForeign, "|- Азиатские сериалы (UHD Video)"),
			port.Map(915, newznab.TVForeign, "|- Корейские сериалы"),
			port.Map(1242, newznab.TVForeign, "|- Корейские сериалы (HD Video)"),
			port.Map(717, newznab.TVForeign, "|- Китайские сериалы"),
			port.Map(1939, newznab.TVForeign, "|- Японские сериалы"),
			port.Map(2412, newznab.TVForeign, "|- Сериалы Таиланда, Индонезии, Сингапура"),
			port.Map(2102, newznab.TVForeign, "|- VMV и др. ролики"),
			port.Map(19, newznab.TVDocumentary, "СМИ"),
			port.Map(670, newznab.TVDocumentary, "Вера и религия"),
			port.Map(1475, newznab.TVDocumentary, "|- [Видео Религия] Христианство"),
			port.Map(2107, newznab.TVDocumentary, "|- [Видео Религия] Ислам"),
			port.Map(1453, newznab.TVDocumentary, "|- [Видео Религия] Культы и новые религиозные движения"),
			port.Map(294, newznab.TVDocumentary, "|- [Видео Религия] Религии Индии, Тибета и Восточной Азии"),
			port.Map(46, newznab.TVDocumentary, "Документальные фильмы и телепередачи"),
			port.Map(103, newznab.TVDocumentary, "|- Документальные (DVD)"),
			port.Map(671, newznab.TVDocumentary, "|- [Док] Биографии. Личности и кумиры"),
			port.Map(2177, newznab.TVDocumentary, "|- [Док] Кинематограф и мультипликация"),
			port.Map(656, newznab.TVDocumentary, "|- [Док] Мастера искусств Театра и Кино"),
			port.Map(2538, newznab.TVDocumentary, "|- [Док] Искусство, история искусств"),
			port.Map(2159, newznab.TVDocumentary, "|- [Док] Музыка"),
			port.Map(251, newznab.TVDocumentary, "|- [Док] Криминальная документалистика"),
			port.Map(98, newznab.TVDocumentary, "|- [Док] Тайны века / Спецслужбы / Теории Заговоров"),
			port.Map(97, newznab.TVDocumentary, "|- [Док] Военное дело"),
			port.Map(851, newznab.TVDocumentary, "|- [Док] Вторая мировая война"),
			port.Map(2178, newznab.TVDocumentary, "|- [Док] Аварии / Катастрофы / Катаклизмы"),
			port.Map(821, newznab.TVDocumentary, "|- [Док] Авиация"),
			port.Map(2076, newznab.TVDocumentary, "|- [Док] Космос"),
			port.Map(56, newznab.TVDocumentary, "|- [Док] Научно-популярные фильмы"),
			port.Map(2123, newznab.TVDocumentary, "|- [Док] Флора и фауна"),
			port.Map(876, newznab.TVDocumentary, "|- [Док] Путешествия и туризм"),
			port.Map(2139, newznab.TVDocumentary, "|- [Док] Медицина"),
			port.Map(2380, newznab.TVDocumentary, "|- [Док] Социальные ток-шоу"),
			port.Map(1467, newznab.TVDocumentary, "|- [Док] Информационно-аналитические и общественно-политические передачи"),
			port.Map(1469, newznab.TVDocumentary, "|- [Док] Архитектура и строительство"),
			port.Map(672, newznab.TVDocumentary, "|- [Док] Всё о доме, быте и дизайне"),
			port.Map(249, newznab.TVDocumentary, "|- [Док] BBC"),
			port.Map(552, newznab.TVDocumentary, "|- [Док] Discovery"),
			port.Map(500, newznab.TVDocumentary, "|- [Док] National Geographic"),
			port.Map(2112, newznab.TVDocumentary, "|- [Док] История: Древний мир / Античность / Средневековье"),
			port.Map(1327, newznab.TVDocumentary, "|- [Док] История: Новое и Новейшее время"),
			port.Map(1468, newznab.TVDocumentary, "|- [Док] Эпоха СССР"),
			port.Map(1280, newznab.TVDocumentary, "|- [Док] Битва экстрасенсов / Теория невероятности / Искатели / Галилео"),
			port.Map(752, newznab.TVDocumentary, "|- [Док] Русские сенсации / Программа Максимум / Профессия репортёр"),
			port.Map(1114, newznab.TVDocumentary, "|- [Док] Паранормальные явления"),
			port.Map(2168, newznab.TVDocumentary, "|- [Док] Альтернативная история и наука"),
			port.Map(2160, newznab.TVDocumentary, "|- [Док] Внежанровая документалистика"),
			port.Map(2176, newznab.TVDocumentary, "|- [Док] Разное / некондиция"),
			port.Map(314, newznab.TVDocumentary, "Документальные (HD Video)"),
			port.Map(2323, newznab.TVDocumentary, "|- Информационно-аналитические и общественно-политические (HD Video)"),
			port.Map(1278, newznab.TVDocumentary, "|- Биографии. Личности и кумиры (HD Video)"),
			port.Map(1281, newznab.TVDocumentary, "|- Военное дело (HD Video)"),
			port.Map(2110, newznab.TVDocumentary, "|- Естествознание, наука и техника (HD Video)"),
			port.Map(979, newznab.TVDocumentary, "|- Путешествия и туризм (HD Video)"),
			port.Map(2169, newznab.TVDocumentary, "|- Флора и фауна (HD Video)"),
			port.Map(2166, newznab.TVDocumentary, "|- История (HD Video)"),
			port.Map(2164, newznab.TVDocumentary, "|- BBC, Discovery, National Geographic, History Channel, Netflix (HD Video)"),
			port.Map(2163, newznab.TVDocumentary, "|- Криминальная документалистика (HD Video)"),
			port.Map(85, newznab.TVDocumentary, "|- Некондиционное видео - Документальные (HD Video)"),
			port.Map(24, newznab.TVDocumentary, "Развлекательные телепередачи и шоу, приколы и юмор"),
			port.Map(1959, newznab.TVOther, "|- [Видео Юмор] Интеллектуальные игры и викторины"),
			port.Map(939, newznab.TVOther, "|- [Видео Юмор] Реалити и ток-шоу / номинации / показы"),
			port.Map(1481, newznab.TVOther, "|- [Видео Юмор] Детские телешоу"),
			port.Map(113, newznab.TVOther, "|- [Видео Юмор] КВН"),
			port.Map(115, newznab.TVOther, "|- [Видео Юмор] Пост КВН"),
			port.Map(882, newznab.TVOther, "|- [Видео Юмор] Кривое Зеркало / Городок / В Городке"),
			port.Map(1482, newznab.TVOther, "|- [Видео Юмор] Ледовые шоу"),
			port.Map(393, newznab.TVOther, "|- [Видео Юмор] Музыкальные шоу"),
			port.Map(1569, newznab.TVOther, "|- [Видео Юмор] Званый ужин"),
			port.Map(373, newznab.TVOther, "|- [Видео Юмор] Хорошие Шутки"),
			port.Map(1186, newznab.TVOther, "|- [Видео Юмор] Вечерний Квартал"),
			port.Map(137, newznab.TVOther, "|- [Видео Юмор] Фильмы со смешным переводом (пародии)"),
			port.Map(2537, newznab.TVOther, "|- [Видео Юмор] Stand-up comedy"),
			port.Map(532, newznab.TVOther, "|- [Видео Юмор] Украинские Шоу"),
			port.Map(827, newznab.TVOther, "|- [Видео Юмор] Танцевальные шоу, концерты, выступления"),
			port.Map(1484, newznab.TVOther, "|- [Видео Юмор] Цирк"),
			port.Map(1485, newznab.TVOther, "|- [Видео Юмор] Школа злословия"),
			port.Map(114, newznab.TVOther, "|- [Видео Юмор] Сатирики и юмористы"),
			port.Map(1332, newznab.TVOther, "|- Юмористические аудиопередачи"),
			port.Map(1495, newznab.TVOther, "|- Аудио и видео ролики (Приколы и юмор)"),
			port.Map(1346, newznab.TVSport, "XXXIII Летние Олимпийские игры 2024"),
			port.Map(2493, newznab.TVSport, "|- Легкая атлетика. Плавание. Прыжки в воду. Синхронное плавание. Гим.."),
			port.Map(2103, newznab.TVSport, "|- Велоспорт. Академическая гребля. Гребля на байдарках и каноэ"),
			port.Map(2485, newznab.TVSport, "|- Футбол. Баскетбол. Волейбол. Гандбол. Водное поло. Регби. Хоккей н.."),
			port.Map(2479, newznab.TVSport, "|- Теннис. Настольный теннис. Бадминтон"),
			port.Map(2089, newznab.TVSport, "|- Фехтование. Стрельба. Стрельба из лука. Современное пятиборье"),
			port.Map(2338, newznab.TVSport, "|- Бокс. Борьба Вольная и Греко-римская. Дзюдо. Карате. Тхэквондо"),
			port.Map(927, newznab.TVSport, "|- Другие виды спорта"),
			port.Map(1392, newznab.TVSport, "XXXII Летние Олимпийские игры 2020"),
			port.Map(2475, newznab.TVSport, "|- Легкая атлетика. Плавание. Прыжки в воду. Синхронное плавание"),
			port.Map(2113, newznab.TVSport, "|- Гимнастика. Прыжки на батуте. Фехтование. Стрельба. Современное пя.."),
			port.Map(2482, newznab.TVSport, "|- Велоспорт. Академическая гребля. Гребля на байдарках и каноэ"),
			port.Map(2522, newznab.TVSport, "|- Бокс. Борьба Вольная и Греко-римская. Дзюдо. Карате. Тхэквондо"),
			port.Map(2486, newznab.TVSport, "|- Баскетбол. Волейбол. Гандбол. Водное поло. Регби. Хоккей на траве"),
			port.Map(1794, newznab.TVSport, "|- Другие виды спорта"),
			port.Map(1315, newznab.TVSport, "XXIV Зимние Олимпийские игры 2022"),
			port.Map(1336, newznab.TVSport, "|- Биатлон"),
			port.Map(2171, newznab.TVSport, "|- Лыжные гонки"),
			port.Map(1339, newznab.TVSport, "|- Прыжки на лыжах с трамплина / Лыжное двоеборье"),
			port.Map(2455, newznab.TVSport, "|- Горные лыжи / Сноубординг / Фристайл"),
			port.Map(1434, newznab.TVSport, "|- Бобслей / Санный спорт / Скелетон"),
			port.Map(2350, newznab.TVSport, "|- Конькобежный спорт / Шорт-трек"),
			port.Map(1472, newznab.TVSport, "|- Фигурное катание"),
			port.Map(2068, newznab.TVSport, "|- Хоккей"),
			port.Map(2016, newznab.TVSport, "|- Керлинг"),
			port.Map(1311, newznab.TVSport, "|- Обзорные и аналитические программы"),
			port.Map(255, newznab.TVSport, "Спортивные турниры, фильмы и передачи"),
			port.Map(256, newznab.TVSport, "|- Автоспорт"),
			port.Map(1986, newznab.TVSport, "|- Мотоспорт"),
			port.Map(660, newznab.TVSport, "|- Формула-1 (2024)"),
			port.Map(1551, newznab.TVSport, "|- Формула-1 (2012-2023)"),
			port.Map(626, newznab.TVSport, "|- Формула 1 (до 2011 вкл.)"),
			port.Map(262, newznab.TVSport, "|- Велоспорт"),
			port.Map(1326, newznab.TVSport, "|- Волейбол/Гандбол"),
			port.Map(978, newznab.TVSport, "|- Бильярд"),
			port.Map(1287, newznab.TVSport, "|- Покер"),
			port.Map(1188, newznab.TVSport, "|- Бодибилдинг/Силовые виды спорта"),
			port.Map(1667, newznab.TVSport, "|- Бокс"),
			port.Map(1675, newznab.TVSport, "|- Классические единоборства"),
			port.Map(257, newznab.TVSport, "|- Смешанные единоборства и K-1"),
			port.Map(875, newznab.TVSport, "|- Американский футбол"),
			port.Map(263, newznab.TVSport, "|- Регби"),
			port.Map(2073, newznab.TVSport, "|- Бейсбол"),
			port.Map(550, newznab.TVSport, "|- Теннис"),
			port.Map(2124, newznab.TVSport, "|- Бадминтон/Настольный теннис"),
			port.Map(1470, newznab.TVSport, "|- Гимнастика/Соревнования по танцам"),
			port.Map(528, newznab.TVSport, "|- Лёгкая атлетика/Водные виды спорта"),
			port.Map(486, newznab.TVSport, "|- Зимние виды спорта"),
			port.Map(854, newznab.TVSport, "|- Фигурное катание"),
			port.Map(2079, newznab.TVSport, "|- Биатлон"),
			port.Map(260, newznab.TVSport, "|- Экстрим"),
			port.Map(1319, newznab.TVSport, "|- Спорт (видео)"),
			port.Map(1608, newznab.TVSport, "⚽ Футбол"),
			port.Map(2294, newznab.TVSport, "|- UHDTV"),
			port.Map(1693, newznab.TVSport, "|- Чемпионат Мира 2026 (отбор)"),
			port.Map(136, newznab.TVSport, "|- Чемпионат Европы 2024 (отбор)"),
			port.Map(2532, newznab.TVSport, "|- Чемпионат Европы 2020 [2021] (финальный турнир)"),
			port.Map(592, newznab.TVSport, "|- Лига Наций"),
			port.Map(1229, newznab.TVSport, "|- Чемпионат Мира 2022"),
			port.Map(2533, newznab.TVSport, "|- Чемпионат Мира 2018 (игры)"),
			port.Map(1952, newznab.TVSport, "|- Чемпионат Мира 2018 (обзорные передачи, документалистика)"),
			port.Map(1621, newznab.TVSport, "|- Чемпионаты Мира"),
			port.Map(2075, newznab.TVSport, "|- Россия 2024-2025"),
			port.Map(1668, newznab.TVSport, "|- Россия 2023-2024"),
			port.Map(1613, newznab.TVSport, "|- Россия/СССР"),
			port.Map(1614, newznab.TVSport, "|- Англия"),
			port.Map(1623, newznab.TVSport, "|- Испания"),
			port.Map(1615, newznab.TVSport, "|- Италия"),
			port.Map(1630, newznab.TVSport, "|- Германия"),
			port.Map(2425, newznab.TVSport, "|- Франция"),
			port.Map(2514, newznab.TVSport, "|- Украина"),
			port.Map(1616, newznab.TVSport, "|- Другие национальные чемпионаты и кубки"),
			port.Map(2014, newznab.TVSport, "|- Международные турниры"),
			port.Map(1442, newznab.TVSport, "|- Еврокубки 2024-2025"),
			port.Map(1491, newznab.TVSport, "|- Еврокубки 2023-2024"),
			port.Map(1987, newznab.TVSport, "|- Еврокубки 2011-2023"),
			port.Map(1617, newznab.TVSport, "|- Еврокубки"),
			port.Map(1620, newznab.TVSport, "|- Чемпионаты Европы"),
			port.Map(1998, newznab.TVSport, "|- Товарищеские турниры и матчи"),
			port.Map(1343, newznab.TVSport, "|- Обзорные и аналитические передачи 2018-2023"),
			port.Map(751, newznab.TVSport, "|- Обзорные и аналитические передачи"),
			port.Map(497, newznab.TVSport, "|- Документальные фильмы (футбол)"),
			port.Map(1697, newznab.TVSport, "|- Мини-футбол/Пляжный футбол"),
			port.Map(2004, newznab.TVSport, "🏀 Баскетбол"),
			port.Map(2001, newznab.TVSport, "|- Международные соревнования"),
			port.Map(2002, newznab.TVSport, "|- NBA / NCAA (до 2000 г.)"),
			port.Map(283, newznab.TVSport, "|- NBA / NCAA (2000-2010 гг.)"),
			port.Map(1997, newznab.TVSport, "|- NBA / NCAA (2010-2024 гг.)"),
			port.Map(2003, newznab.TVSport, "|- Европейский клубный баскетбол"),
			port.Map(2009, newznab.TVSport, "🏒 Хоккей"),
			port.Map(2010, newznab.TVSport, "|- Хоккей с мячом / Бенди"),
			port.Map(2006, newznab.TVSport, "|- Международные турниры"),
			port.Map(2007, newznab.TVSport, "|- КХЛ"),
			port.Map(2005, newznab.TVSport, "|- НХЛ (до 2011/12)"),
			port.Map(259, newznab.TVSport, "|- НХЛ (с 2013)"),
			port.Map(2008, newznab.TVSport, "|- СССР - Канада"),
			port.Map(126, newznab.TVSport, "|- Документальные фильмы и аналитика"),
			port.Map(845, newznab.TVSport, "Рестлинг"),
			port.Map(343, newznab.TVSport, "|- Professional Wrestling"),
			port.Map(2111, newznab.TVSport, "|- Independent Wrestling"),
			port.Map(1527, newznab.TVSport, "|- International Wrestling"),
			port.Map(2069, newznab.TVSport, "|- Oldschool Wrestling"),
			port.Map(1323, newznab.TVSport, "|- Documentary Wrestling"),
			port.Map(1411, newznab.Books, "|- Сканирование, обработка сканов"),
			port.Map(21, newznab.Books, "Книги и журналы (общий раздел)"),
			port.Map(2157, newznab.Books, "|- Кино, театр, ТВ, мультипликация, цирк"),
			port.Map(765, newznab.Books, "|- Рисунок, графический дизайн"),
			port.Map(2019, newznab.Books, "|- Фото и видеосъемка"),
			port.Map(31, newznab.BooksMags, "|- Журналы и газеты (общий раздел)"),
			port.Map(1427, newznab.Books, "|- Эзотерика, гадания, магия, фен-шуй"),
			port.Map(2422, newznab.Books, "|- Астрология"),
			port.Map(2195, newznab.Books, "|- Красота. Уход. Домоводство"),
			port.Map(2521, newznab.Books, "|- Мода. Стиль. Этикет"),
			port.Map(2223, newznab.Books, "|- Путешествия и туризм"),
			port.Map(2447, newznab.Books, "|- Знаменитости и кумиры"),
			port.Map(39, newznab.Books, "|- Разное (книги)"),
			port.Map(2086, newznab.Books, "|- Самиздат, статьи из журналов, фрагменты книг"),
			port.Map(1101, newznab.Books, "Для детей, родителей и учителей"),
			port.Map(745, newznab.Books, "|- Учебная литература для детского сада и начальной школы (до 4 класса)"),
			port.Map(1689, newznab.Books, "|- Учебная литература для старших классов (5-11 класс)"),
			port.Map(2336, newznab.Books, "|- Учителям и педагогам"),
			port.Map(2337, newznab.Books, "|- Научно-популярная и познавательная литература (для детей)"),
			port.Map(1353, newznab.Books, "|- Досуг и творчество"),
			port.Map(1400, newznab.Books, "|- Воспитание и развитие"),
			port.Map(1415, newznab.Books, "|- Худ. лит-ра для дошкольников и младших классов"),
			port.Map(2046, newznab.Books, "|- Худ. лит-ра для средних и старших классов"),
			port.Map(1802, newznab.Books, "Спорт, физическая культура, боевые искусства"),
			port.Map(2189, newznab.Books, "|- Футбол (книги и журналы)"),
			port.Map(2190, newznab.Books, "|- Хоккей (книги и журналы)"),
			port.Map(2443, newznab.Books, "|- Игровые виды спорта"),
			port.Map(1477, newznab.Books, "|- Легкая атлетика. Плавание. Гимнастика. Тяжелая атлетика. Гребля"),
			port.Map(669, newznab.Books, "|- Автоспорт. Мотоспорт. Велоспорт"),
			port.Map(2196, newznab.Books, "|- Шахматы. Шашки"),
			port.Map(2056, newznab.Books, "|- Боевые искусства, единоборства"),
			port.Map(1436, newznab.Books, "|- Экстрим (книги)"),
			port.Map(2191, newznab.Books, "|- Физкультура, фитнес, бодибилдинг"),
			port.Map(2477, newznab.Books, "|- Спортивная пресса"),
			port.Map(1680, newznab.Books, "Гуманитарные науки"),
			port.Map(1684, newznab.Books, "|- Искусствоведение. Культурология"),
			port.Map(2446, newznab.Books, "|- Фольклор. Эпос. Мифология"),
			port.Map(2524, newznab.Books, "|- Литературоведение"),
			port.Map(2525, newznab.Books, "|- Лингвистика"),
			port.Map(995, newznab.Books, "|- Философия"),
			port.Map(2022, newznab.Books, "|- Политология"),
			port.Map(2471, newznab.Books, "|- Социология"),
			port.Map(2375, newznab.Books, "|- Публицистика, журналистика"),
			port.Map(764, newznab.Books, "|- Бизнес, менеджмент"),
			port.Map(1685, newznab.Books, "|- Маркетинг"),
			port.Map(1688, newznab.Books, "|- Экономика"),
			port.Map(2472, newznab.Books, "|- Финансы"),
			port.Map(1687, newznab.Books, "|- Юридические науки. Право. Криминалистика"),
			port.Map(2020, newznab.Books, "Исторические науки"),
			port.Map(1349, newznab.Books, "|- Методология и философия исторической науки"),
			port.Map(1967, newznab.Books, "|- Исторические источники (книги, периодика)"),
			port.Map(1341, newznab.Books, "|- Исторические источники (документы)"),
			port.Map(2049, newznab.Books, "|- Исторические персоны"),
			port.Map(1681, newznab.Books, "|- Альтернативные исторические теории"),
			port.Map(2319, newznab.Books, "|- Археология"),
			port.Map(2434, newznab.Books, "|- Древний мир. Античность"),
			port.Map(1683, newznab.Books, "|- Средние века"),
			port.Map(2444, newznab.Books, "|- История Нового и Новейшего времени"),
			port.Map(2427, newznab.Books, "|- История Европы"),
			port.Map(2452, newznab.Books, "|- История Азии и Африки"),
			port.Map(2445, newznab.Books, "|- История Америки, Австралии, Океании"),
			port.Map(2435, newznab.Books, "|- История России"),
			port.Map(667, newznab.Books, "|- История России до 1917 года"),
			port.Map(2436, newznab.Books, "|- Эпоха СССР"),
			port.Map(1335, newznab.Books, "|- История России после 1991 года"),
			port.Map(2453, newznab.Books, "|- История стран бывшего СССР"),
			port.Map(2320, newznab.Books, "|- Этнография, антропология"),
			port.Map(1801, newznab.Books, "|- Международные отношения. Дипломатия"),
			port.Map(2023, newznab.BooksTechnical, "Точные, естественные и инженерные науки"),
			port.Map(2024, newznab.BooksTechnical, "|- Авиация / Космонавтика"),
			port.Map(2026, newznab.BooksTechnical, "|- Физика"),
			port.Map(2192, newznab.BooksTechnical, "|- Астрономия"),
			port.Map(2027, newznab.BooksTechnical, "|- Биология / Экология"),
			port.Map(295, newznab.BooksTechnical, "|- Химия / Биохимия"),
			port.Map(2028, newznab.BooksTechnical, "|- Математика"),
			port.Map(2029, newznab.BooksTechnical, "|- География / Геология / Геодезия"),
			port.Map(1325, newznab.BooksTechnical, "|- Электроника / Радио"),
			port.Map(2386, newznab.BooksTechnical, "|- Схемы и сервис-мануалы (оригинальная документация)"),
			port.Map(2031, newznab.BooksTechnical, "|- Архитектура / Строительство / Инженерные сети / Ландшафтный дизайн"),
			port.Map(2030, newznab.BooksTechnical, "|- Машиностроение"),
			port.Map(2526, newznab.BooksTechnical, "|- Сварка / Пайка / Неразрушающий контроль"),
			port.Map(2527, newznab.BooksTechnical, "|- Автоматизация / Робототехника"),
			port.Map(2254, newznab.BooksTechnical, "|- Металлургия / Материаловедение"),
			port.Map(2376, newznab.BooksTechnical, "|- Механика, сопротивление материалов"),
			port.Map(2054, newznab.BooksTechnical, "|- Энергетика / электротехника"),
			port.Map(770, newznab.BooksTechnical, "|- Нефтяная, газовая и химическая промышленность"),
			port.Map(2476, newznab.BooksTechnical, "|- Сельское хозяйство и пищевая промышленность"),
			port.Map(2494, newznab.BooksTechnical, "|- Железнодорожное дело"),
			port.Map(1528, newznab.BooksTechnical, "|- Нормативная документация"),
			port.Map(2032, newznab.BooksTechnical, "|- Журналы: научные, научно-популярные, радио и др."),
			port.Map(919, newznab.Books, "Ноты и Музыкальная литература"),
			port.Map(944, newznab.Books, "|- Академическая музыка (Ноты и Media CD)"),
			port.Map(980, newznab.Books, "|- Другие направления (Ноты, табулатуры)"),
			port.Map(946, newznab.Books, "|- Самоучители и Школы"),
			port.Map(977, newznab.Books, "|- Песенники (Songbooks)"),
			port.Map(2074, newznab.Books, "|- Музыкальная литература и Теория"),
			port.Map(2349, newznab.Books, "|- Музыкальные журналы"),
			port.Map(768, newznab.Books, "Военное дело"),
			port.Map(2099, newznab.Books, "|- Милитария"),
			port.Map(2021, newznab.Books, "|- Военная история"),
			port.Map(2437, newznab.Books, "|- История Второй мировой войны"),
			port.Map(1337, newznab.Books, "|- Биографии и мемуары военных деятелей"),
			port.Map(1447, newznab.Books, "|- Военная техника"),
			port.Map(2468, newznab.Books, "|- Стрелковое оружие"),
			port.Map(2469, newznab.Books, "|- Учебно-методическая литература"),
			port.Map(2470, newznab.Books, "|- Спецслужбы мира"),
			port.Map(1686, newznab.Books, "Вера и религия"),
			port.Map(2215, newznab.Books, "|- Христианство"),
			port.Map(2216, newznab.Books, "|- Ислам"),
			port.Map(2217, newznab.Books, "|- Религии Индии, Тибета и Восточной Азии / Иудаизм"),
			port.Map(2218, newznab.Books, "|- Нетрадиционные религиозные, духовные и мистические учения"),
			port.Map(2252, newznab.Books, "|- Религиоведение. История Религии"),
			port.Map(2543, newznab.Books, "|- Атеизм. Научный атеизм"),
			port.Map(767, newznab.Books, "Психология"),
			port.Map(2515, newznab.Books, "|- Общая и прикладная психология"),
			port.Map(2516, newznab.Books, "|- Психотерапия и консультирование"),
			port.Map(2517, newznab.Books, "|- Психодиагностика и психокоррекция"),
			port.Map(2518, newznab.Books, "|- Социальная психология и психология отношений"),
			port.Map(2519, newznab.Books, "|- Тренинг и коучинг"),
			port.Map(2520, newznab.Books, "|- Саморазвитие и самосовершенствование"),
			port.Map(1696, newznab.Books, "|- Популярная психология"),
			port.Map(2253, newznab.Books, "|- Сексология. Взаимоотношения полов (18+)"),
			port.Map(2033, newznab.Books, "Коллекционирование, увлечения и хобби"),
			port.Map(1412, newznab.Books, "|- Коллекционирование и вспомогательные ист. дисциплины"),
			port.Map(1446, newznab.Books, "|- Вышивание"),
			port.Map(753, newznab.Books, "|- Вязание"),
			port.Map(2037, newznab.Books, "|- Шитье, пэчворк"),
			port.Map(2224, newznab.Books, "|- Кружевоплетение"),
			port.Map(2194, newznab.Books, "|- Бисероплетение. Ювелирика. Украшения из проволоки."),
			port.Map(2418, newznab.Books, "|- Бумажный арт"),
			port.Map(1410, newznab.Books, "|- Другие виды декоративно-прикладного искусства"),
			port.Map(2034, newznab.Books, "|- Домашние питомцы и аквариумистика"),
			port.Map(2433, newznab.Books, "|- Охота и рыбалка"),
			port.Map(1961, newznab.Books, "|- Кулинария (книги)"),
			port.Map(2432, newznab.Books, "|- Кулинария (газеты и журналы)"),
			port.Map(565, newznab.Books, "|- Моделизм"),
			port.Map(1523, newznab.Books, "|- Приусадебное хозяйство / Цветоводство"),
			port.Map(1575, newznab.Books, "|- Ремонт, частное строительство, дизайн интерьеров"),
			port.Map(1520, newznab.Books, "|- Деревообработка"),
			port.Map(2424, newznab.Books, "|- Настольные игры"),
			port.Map(769, newznab.Books, "|- Прочие хобби и игры"),
			port.Map(2038, newznab.Books, "Художественная литература"),
			port.Map(2043, newznab.Books, "|- Русская литература"),
			port.Map(2042, newznab.Books, "|- Зарубежная литература (до 1900 г.)"),
			port.Map(2041, newznab.Books, "|- Зарубежная литература (XX и XXI век)"),
			port.Map(2044, newznab.Books, "|- Детектив, боевик"),
			port.Map(2039, newznab.Books, "|- Женский роман"),
			port.Map(2045, newznab.Books, "|- Отечественная фантастика / фэнтези / мистика"),
			port.Map(2080, newznab.Books, "|- Зарубежная фантастика / фэнтези / мистика"),
			port.Map(2047, newznab.Books, "|- Приключения"),
			port.Map(2193, newznab.Books, "|- Литературные журналы"),
			port.Map(1037, newznab.Books, "|- Самиздат и книги, изданные за счет авторов"),
			port.Map(1418, newznab.BooksTechnical, "Компьютерная литература"),
			port.Map(1422, newznab.BooksTechnical, "|- Программы от Microsoft"),
			port.Map(1423, newznab.BooksTechnical, "|- Другие программы"),
			port.Map(1424, newznab.BooksTechnical, "|- Mac OS; Linux, FreeBSD и прочие *NIX"),
			port.Map(1445, newznab.BooksTechnical, "|- СУБД"),
			port.Map(1425, newznab.BooksTechnical, "|- Веб-дизайн и программирование"),
			port.Map(1426, newznab.BooksTechnical, "|- Программирование (книги)"),
			port.Map(1428, newznab.BooksTechnical, "|- Графика, обработка видео"),
			port.Map(1429, newznab.BooksTechnical, "|- Сети / VoIP"),
			port.Map(1430, newznab.BooksTechnical, "|- Хакинг и безопасность"),
			port.Map(1431, newznab.BooksTechnical, "|- Железо (книги о ПК)"),
			port.Map(1433, newznab.BooksTechnical, "|- Инженерные и научные программы (книги)"),
			port.Map(1432, newznab.BooksTechnical, "|- Компьютерные журналы и приложения к ним"),
			port.Map(2202, newznab.BooksTechnical, "|- Дисковые приложения к игровым журналам"),
			port.Map(862, newznab.BooksComics, "Комиксы, манга, ранобэ"),
			port.Map(2461, newznab.BooksComics, "|- Комиксы на русском языке"),
			port.Map(2462, newznab.BooksComics, "|- Комиксы издательства Marvel"),
			port.Map(2463, newznab.BooksComics, "|- Комиксы издательства DC"),
			port.Map(2464, newznab.BooksComics, "|- Комиксы других издательств"),
			port.Map(2473, newznab.BooksComics, "|- Комиксы на других языках"),
			port.Map(281, newznab.BooksComics, "|- Манга (на русском языке)"),
			port.Map(2465, newznab.BooksComics, "|- Манга (на иностранных языках)"),
			port.Map(2458, newznab.BooksComics, "|- Ранобэ"),
			port.Map(2048, newznab.BooksOther, "Коллекции книг и библиотеки"),
			port.Map(1238, newznab.BooksOther, "|- Библиотеки (зеркала сетевых библиотек/коллекций)"),
			port.Map(2055, newznab.BooksOther, "|- Тематические коллекции (подборки)"),
			port.Map(754, newznab.BooksOther, "|- Многопредметные коллекции (подборки)"),
			port.Map(2114, newznab.BooksEBook, "Мультимедийные и интерактивные издания"),
			port.Map(2438, newznab.BooksEBook, "|- Мультимедийные энциклопедии"),
			port.Map(2439, newznab.BooksEBook, "|- Интерактивные обучающие и развивающие материалы"),
			port.Map(2440, newznab.BooksEBook, "|- Обучающие издания для детей"),
			port.Map(2441, newznab.BooksEBook, "|- Кулинария. Цветоводство. Домоводство"),
			port.Map(2442, newznab.BooksEBook, "|- Культура. Искусство. История"),
			port.Map(2125, newznab.Books, "Медицина и здоровье"),
			port.Map(2133, newznab.Books, "|- Клиническая медицина до 1980 год"),
			port.Map(2130, newznab.Books, "|- Клиническая медицина с 1980 по 2000 год"),
			port.Map(2313, newznab.Books, "|- Клиническая медицина после 2000 год"),
			port.Map(2528, newznab.Books, "|- Научная медицинская периодика (газеты и журналы)"),
			port.Map(2129, newznab.Books, "|- Медико-биологические науки"),
			port.Map(2141, newznab.Books, "|- Фармация и фармакология"),
			port.Map(2314, newznab.Books, "|- Популярная медицинская периодика (газеты и журналы)"),
			port.Map(2132, newznab.Books, "|- Нетрадиционная, народная медицина и популярные книги о здоровье"),
			port.Map(2131, newznab.Books, "|- Ветеринария, разное"),
			port.Map(2315, newznab.Books, "|- Тематические коллекции книг"),
			port.Map(2362, newznab.BooksEBook, "Иностранные языки для взрослых"),
			port.Map(1265, newznab.BooksEBook, "|- Английский язык (для взрослых)"),
			port.Map(1266, newznab.BooksEBook, "|- Немецкий язык"),
			port.Map(1267, newznab.BooksEBook, "|- Французский язык"),
			port.Map(1358, newznab.BooksEBook, "|- Испанский язык"),
			port.Map(2363, newznab.BooksEBook, "|- Итальянский язык"),
			port.Map(734, newznab.BooksEBook, "|- Финский язык"),
			port.Map(1268, newznab.BooksEBook, "|- Другие европейские языки"),
			port.Map(1673, newznab.BooksEBook, "|- Арабский язык"),
			port.Map(1269, newznab.BooksEBook, "|- Китайский язык"),
			port.Map(1270, newznab.BooksEBook, "|- Японский язык"),
			port.Map(1275, newznab.BooksEBook, "|- Другие восточные языки"),
			port.Map(2364, newznab.BooksEBook, "|- Русский язык как иностранный"),
			port.Map(1276, newznab.BooksEBook, "|- Мультиязычные сборники и курсы"),
			port.Map(2094, newznab.BooksEBook, "|- LIM-курсы"),
			port.Map(1274, newznab.BooksEBook, "|- Разное (иностранные языки)"),
			port.Map(1264, newznab.BooksEBook, "Иностранные языки для детей"),
			port.Map(2358, newznab.BooksEBook, "|- Английский язык (для детей)"),
			port.Map(2359, newznab.BooksEBook, "|- Другие европейские языки (для детей)"),
			port.Map(2360, newznab.BooksEBook, "|- Восточные языки (для детей)"),
			port.Map(2361, newznab.BooksEBook, "|- Школьные учебники, ЕГЭ, ОГЭ"),
			port.Map(2057, newznab.BooksEBook, "Художественная литература (ин.языки)"),
			port.Map(2355, newznab.BooksEBook, "|- Художественная литература на английском языке"),
			port.Map(2474, newznab.BooksEBook, "|- Художественная литература на французском языке"),
			port.Map(2356, newznab.BooksEBook, "|- Художественная литература на других европейских языках"),
			port.Map(2357, newznab.BooksEBook, "|- Художественная литература на восточных языках"),
			port.Map(2413, newznab.AudioAudiobook, "Аудиокниги на иностранных языках"),
			port.Map(1501, newznab.AudioAudiobook, "|- Аудиокниги на английском языке"),
			port.Map(1580, newznab.AudioAudiobook, "|- Аудиокниги на немецком языке"),
			port.Map(525, newznab.AudioAudiobook, "|- Аудиокниги на других иностранных языках"),
			port.Map(610, newznab.BooksOther, "Видеоуроки и обучающие интерактивные DVD"),
			port.Map(1568, newznab.BooksOther, "|- Кулинария"),
			port.Map(1542, newznab.BooksOther, "|- Спорт"),
			port.Map(2335, newznab.BooksOther, "|- Фитнес - Кардио-Силовые Тренировки"),
			port.Map(1544, newznab.BooksOther, "|- Фитнес - Разум и Тело"),
			port.Map(1546, newznab.BooksOther, "|- Бодибилдинг"),
			port.Map(1549, newznab.BooksOther, "|- Оздоровительные практики"),
			port.Map(1597, newznab.BooksOther, "|- Йога"),
			port.Map(1552, newznab.BooksOther, "|- Видео- и фотосъёмка"),
			port.Map(1550, newznab.BooksOther, "|- Уход за собой"),
			port.Map(1553, newznab.BooksOther, "|- Рисование"),
			port.Map(1554, newznab.BooksOther, "|- Игра на гитаре"),
			port.Map(617, newznab.BooksOther, "|- Ударные инструменты"),
			port.Map(1555, newznab.BooksOther, "|- Другие музыкальные инструменты"),
			port.Map(2017, newznab.BooksOther, "|- Игра на бас-гитаре"),
			port.Map(1257, newznab.BooksOther, "|- Бальные танцы"),
			port.Map(1258, newznab.BooksOther, "|- Танец живота"),
			port.Map(2208, newznab.BooksOther, "|- Уличные и клубные танцы"),
			port.Map(677, newznab.BooksOther, "|- Танцы, разное"),
			port.Map(1255, newznab.BooksOther, "|- Охота"),
			port.Map(1479, newznab.BooksOther, "|- Рыболовство и подводная охота"),
			port.Map(1261, newznab.BooksOther, "|- Фокусы и трюки"),
			port.Map(614, newznab.BooksOther, "|- Образование"),
			port.Map(1583, newznab.BooksOther, "|- Финансы"),
			port.Map(1259, newznab.BooksOther, "|- Продажи, бизнес"),
			port.Map(2065, newznab.BooksOther, "|- Беременность, роды, материнство"),
			port.Map(1254, newznab.BooksOther, "|- Учебные видео для детей"),
			port.Map(1260, newznab.BooksOther, "|- Психология"),
			port.Map(2209, newznab.BooksOther, "|- Эзотерика, саморазвитие"),
			port.Map(2210, newznab.BooksOther, "|- Пикап, знакомства"),
			port.Map(1547, newznab.BooksOther, "|- Строительство, ремонт и дизайн"),
			port.Map(1548, newznab.BooksOther, "|- Дерево- и металлообработка"),
			port.Map(2211, newznab.BooksOther, "|- Растения и животные"),
			port.Map(1596, newznab.BooksOther, "|- Хобби и рукоделие"),
			port.Map(2135, newznab.BooksOther, "|- Медицина и стоматология"),
			port.Map(2140, newznab.BooksOther, "|- Психотерапия и клиническая психология"),
			port.Map(2136, newznab.BooksOther, "|- Массаж"),
			port.Map(2138, newznab.BooksOther, "|- Здоровье"),
			port.Map(615, newznab.BooksOther, "|- Разное"),
			port.Map(1581, newznab.BooksOther, "Боевые искусства (Видеоуроки)"),
			port.Map(1590, newznab.BooksOther, "|- Айкидо и айки-дзюцу"),
			port.Map(1587, newznab.BooksOther, "|- Вин чун"),
			port.Map(1594, newznab.BooksOther, "|- Джиу-джитсу"),
			port.Map(1591, newznab.BooksOther, "|- Дзюдо и самбо"),
			port.Map(1588, newznab.BooksOther, "|- Каратэ"),
			port.Map(1585, newznab.BooksOther, "|- Работа с оружием"),
			port.Map(1586, newznab.BooksOther, "|- Русский стиль"),
			port.Map(2078, newznab.BooksOther, "|- Рукопашный бой"),
			port.Map(1929, newznab.BooksOther, "|- Смешанные стили"),
			port.Map(1593, newznab.BooksOther, "|- Ударные стили"),
			port.Map(1592, newznab.BooksOther, "|- Ушу"),
			port.Map(1595, newznab.BooksOther, "|- Разное"),
			port.Map(1556, newznab.BooksTechnical, "Компьютерные видеоуроки и обучающие интерактивные DVD"),
			port.Map(2539, newznab.BooksTechnical, "|- Machine/Deep Learning, Neural Networks"),
			port.Map(1560, newznab.BooksTechnical, "|- Компьютерные сети и безопасность"),
			port.Map(1991, newznab.BooksTechnical, "|- Devops"),
			port.Map(1561, newznab.BooksTechnical, "|- ОС и серверные программы Microsoft"),
			port.Map(1653, newznab.BooksTechnical, "|- Офисные программы Microsoft"),
			port.Map(1570, newznab.BooksTechnical, "|- ОС и программы семейства UNIX"),
			port.Map(1654, newznab.BooksTechnical, "|- Adobe Photoshop"),
			port.Map(1655, newznab.BooksTechnical, "|- Autodesk Maya"),
			port.Map(1656, newznab.BooksTechnical, "|- Autodesk 3ds Max"),
			port.Map(1930, newznab.BooksTechnical, "|- Autodesk Softimage (XSI)"),
			port.Map(1931, newznab.BooksTechnical, "|- ZBrush"),
			port.Map(1932, newznab.BooksTechnical, "|- Flash, Flex и ActionScript"),
			port.Map(1562, newznab.BooksTechnical, "|- 2D-графика"),
			port.Map(1563, newznab.BooksTechnical, "|- 3D-графика"),
			port.Map(1626, newznab.BooksTechnical, "|- Инженерные и научные программы (видеоуроки)"),
			port.Map(1564, newznab.BooksTechnical, "|- Web-дизайн"),
			port.Map(1545, newznab.BooksTechnical, "|- WEB, SMM, SEO, интернет-маркетинг"),
			port.Map(1565, newznab.BooksTechnical, "|- Программирование (видеоуроки)"),
			port.Map(1559, newznab.BooksTechnical, "|- Программы для Mac OS"),
			port.Map(1566, newznab.BooksTechnical, "|- Работа с видео"),
			port.Map(1573, newznab.BooksTechnical, "|- Работа со звуком"),
			port.Map(1567, newznab.BooksTechnical, "|- Разное (Компьютерные видеоуроки)"),
			port.Map(2326, newznab.AudioAudiobook, "Радиоспектакли, история, мемуары"),
			port.Map(574, newznab.AudioAudiobook, "|- [Аудио] Радиоспектакли и литературные чтения"),
			port.Map(1036, newznab.AudioAudiobook, "|- [Аудио] Биографии и мемуары"),
			port.Map(400, newznab.AudioAudiobook, "|- [Аудио] История, культурология, философия"),
			port.Map(2389, newznab.AudioAudiobook, "Фантастика, фэнтези, мистика, ужасы, фанфики"),
			port.Map(2388, newznab.AudioAudiobook, "|- [Аудио] Зарубежная фантастика, фэнтези, мистика, ужасы, фанфики"),
			port.Map(2387, newznab.AudioAudiobook, "|- [Аудио] Российская фантастика, фэнтези, мистика, ужасы, фанфики"),
			port.Map(661, newznab.AudioAudiobook, "|- [Аудио] Любовно-фантастический роман"),
			port.Map(2348, newznab.AudioAudiobook, "|- [Аудио] Сборники/разное Фантастика, фэнтези, мистика, ужасы, фанфики"),
			port.Map(2327, newznab.AudioAudiobook, "Художественная литература"),
			port.Map(695, newznab.AudioAudiobook, "|- [Аудио] Поэзия"),
			port.Map(399, newznab.AudioAudiobook, "|- [Аудио] Зарубежная литература"),
			port.Map(402, newznab.AudioAudiobook, "|- [Аудио] Русская литература"),
			port.Map(467, newznab.AudioAudiobook, "|- [Аудио] Современные любовные романы"),
			port.Map(490, newznab.AudioAudiobook, "|- [Аудио] Детская литература"),
			port.Map(499, newznab.AudioAudiobook, "|- [Аудио] Зарубежные детективы, приключения, триллеры, боевики"),
			port.Map(2137, newznab.AudioAudiobook, "|- [Аудио] Российские детективы, приключения, триллеры, боевики"),
			port.Map(2127, newznab.AudioAudiobook, "|- [Аудио] Азиатская подростковая литература, ранобэ, веб-новеллы"),
			port.Map(2324, newznab.AudioAudiobook, "Религии"),
			port.Map(2325, newznab.AudioAudiobook, "|- [Аудио] Православие"),
			port.Map(2342, newznab.AudioAudiobook, "|- [Аудио] Ислам"),
			port.Map(530, newznab.AudioAudiobook, "|- [Аудио] Другие традиционные религии"),
			port.Map(2152, newznab.AudioAudiobook, "|- [Аудио] Нетрадиционные религиозно-философские учения"),
			port.Map(2328, newznab.AudioAudiobook, "Прочая литература"),
			port.Map(1350, newznab.AudioAudiobook, "|- [Аудио] Книги по медицине"),
			port.Map(403, newznab.AudioAudiobook, "|- [Аудио] Учебная и научно-популярная литература"),
			port.Map(1279, newznab.AudioAudiobook, "|- [Аудио] lossless-аудиокниги"),
			port.Map(716, newznab.AudioAudiobook, "|- [Аудио] Бизнес"),
			port.Map(2165, newznab.AudioAudiobook, "|- [Аудио] Разное"),
			port.Map(401, newznab.AudioAudiobook, "|- [Аудио] Некондиционные раздачи"),
			port.Map(1964, newznab.Books, "Ремонт и эксплуатация транспортных средств"),
			port.Map(1973, newznab.Books, "|- Оригинальные каталоги по подбору запчастей"),
			port.Map(1974, newznab.Books, "|- Неоригинальные каталоги по подбору запчастей"),
			port.Map(1975, newznab.Books, "|- Программы по диагностике и ремонту"),
			port.Map(1976, newznab.Books, "|- Тюнинг, чиптюнинг, настройка"),
			port.Map(1977, newznab.Books, "|- Книги по ремонту/обслуживанию/эксплуатации ТС"),
			port.Map(1203, newznab.Books, "|- Мультимедийки по ремонту/обслуживанию/эксплуатации ТС"),
			port.Map(1978, newznab.Books, "|- Учет, утилиты и прочее"),
			port.Map(1979, newznab.Books, "|- Виртуальная автошкола"),
			port.Map(1980, newznab.Books, "|- Видеоуроки по вождению транспортных средств"),
			port.Map(1981, newznab.Books, "|- Видеоуроки по ремонту транспортных средств"),
			port.Map(1970, newznab.Books, "|- Журналы по авто/мото"),
			port.Map(334, newznab.Books, "|- Водный транспорт"),
			port.Map(1202, newznab.TVDocumentary, "Фильмы и передачи по авто/мото"),
			port.Map(1985, newznab.TVDocumentary, "|- Документальные/познавательные фильмы"),
			port.Map(1982, newznab.TVOther, "|- Развлекательные передачи"),
			port.Map(2151, newznab.TVDocumentary, "|- Top Gear/Топ Гир"),
			port.Map(1983, newznab.TVDocumentary, "|- Тест драйв/Обзоры/Автосалоны"),
			port.Map(1984, newznab.TVDocumentary, "|- Тюнинг/форсаж"),
			port.Map(409, newznab.Audio, "Классическая и современная академическая музыка"),
			port.Map(560, newznab.AudioLossless, "|- Полные собрания сочинений и многодисковые издания (lossless)"),
			port.Map(794, newznab.AudioLossless, "|- Опера (lossless)"),
			port.Map(556, newznab.AudioLossless, "|- Вокальная музыка (lossless)"),
			port.Map(2307, newznab.AudioLossless, "|- Хоровая музыка (lossless)"),
			port.Map(557, newznab.AudioLossless, "|- Оркестровая музыка (lossless)"),
			port.Map(2308, newznab.AudioLossless, "|- Концерт для инструмента с оркестром (lossless)"),
			port.Map(558, newznab.AudioLossless, "|- Камерная инструментальная музыка (lossless)"),
			port.Map(793, newznab.AudioLossless, "|- Сольная инструментальная музыка (lossless)"),
			port.Map(1395, newznab.AudioLossless, "|- Духовные песнопения и музыка (lossless)"),
			port.Map(1396, newznab.AudioMP3, "|- Духовные песнопения и музыка (lossy)"),
			port.Map(436, newznab.AudioMP3, "|- Полные собрания сочинений и многодисковые издания (lossy)"),
			port.Map(2309, newznab.AudioMP3, "|- Вокальная и хоровая музыка (lossy)"),
			port.Map(2310, newznab.AudioMP3, "|- Оркестровая музыка (lossy)"),
			port.Map(2311, newznab.AudioMP3, "|- Камерная и сольная инструментальная музыка (lossy)"),
			port.Map(969, newznab.Audio, "|- Классика в современной обработке, Classical Crossover (lossy и lossless)"),
			port.Map(1125, newznab.Audio, "Фольклор, Народная и Этническая музыка"),
			port.Map(1130, newznab.AudioMP3, "|- Восточноевропейский фолк (lossy)"),
			port.Map(1131, newznab.AudioLossless, "|- Восточноевропейский фолк (lossless)"),
			port.Map(1132, newznab.AudioMP3, "|- Западноевропейский фолк (lossy)"),
			port.Map(1133, newznab.AudioLossless, "|- Западноевропейский фолк (lossless)"),
			port.Map(2084, newznab.Audio, "|- Klezmer и Еврейский фольклор (lossy и lossless)"),
			port.Map(1128, newznab.AudioMP3, "|- Этническая музыка Сибири, Средней и Восточной Азии (lossy)"),
			port.Map(1129, newznab.AudioLossless, "|- Этническая музыка Сибири, Средней и Восточной Азии (lossless)"),
			port.Map(1856, newznab.AudioMP3, "|- Этническая музыка Индии (lossy)"),
			port.Map(2430, newznab.AudioLossless, "|- Этническая музыка Индии (lossless)"),
			port.Map(1283, newznab.AudioMP3, "|- Этническая музыка Африки и Ближнего Востока (lossy)"),
			port.Map(2085, newznab.AudioLossless, "|- Этническая музыка Африки и Ближнего Востока (lossless)"),
			port.Map(1282, newznab.Audio, "|- Фольклорная, Народная, Эстрадная музыка Кавказа и Закавказья (lossy и lossless)"),
			port.Map(1284, newznab.AudioMP3, "|- Этническая музыка Северной и Южной Америки (lossy)"),
			port.Map(1285, newznab.AudioLossless, "|- Этническая музыка Северной и Южной Америки (lossless)"),
			port.Map(1138, newznab.Audio, "|- Этническая музыка Австралии, Тихого и Индийского океанов (lossy и lossless)"),
			port.Map(1136, newznab.AudioMP3, "|- Country, Bluegrass (lossy)"),
			port.Map(1137, newznab.AudioLossless, "|- Country, Bluegrass (lossless)"),
			port.Map(1849, newznab.Audio, "New Age, Relax, Meditative & Flamenco"),
			port.Map(1126, newznab.AudioMP3, "|- New Age & Meditative (lossy)"),
			port.Map(1127, newznab.AudioLossless, "|- New Age & Meditative (lossless)"),
			port.Map(1134, newznab.AudioMP3, "|- Фламенко и акустическая гитара (lossy)"),
			port.Map(1135, newznab.AudioLossless, "|- Фламенко и акустическая гитара (lossless)"),
			port.Map(2018, newznab.Audio, "|- Музыка для бальных танцев (lossy и lossless)"),
			port.Map(855, newznab.Audio, "|- Звуки природы"),
			port.Map(408, newznab.Audio, "Рэп, Хип-Хоп, R'n'B"),
			port.Map(441, newznab.AudioMP3, "|- Отечественный Рэп, Хип-Хоп (lossy)"),
			port.Map(1173, newznab.AudioMP3, "|- Отечественный R'n'B (lossy)"),
			port.Map(1486, newznab.AudioLossless, "|- Отечественный Рэп, Хип-Хоп, R'n'B (lossless)"),
			port.Map(1172, newznab.AudioMP3, "|- Зарубежный R'n'B (lossy)"),
			port.Map(446, newznab.AudioMP3, "|- Зарубежный Рэп, Хип-Хоп (lossy)"),
			port.Map(909, newznab.AudioLossless, "|- Зарубежный Рэп, Хип-Хоп (lossless)"),
			port.Map(1665, newznab.AudioLossless, "|- Зарубежный R'n'B (lossless)"),
			port.Map(1760, newznab.Audio, "Reggae, Ska, Dub"),
			port.Map(1764, newznab.Audio, "|- Rocksteady, Early Reggae, Ska-Jazz, Trad.Ska (lossy и lossless)"),
			port.Map(1767, newznab.AudioMP3, "|- 3rd Wave Ska (lossy)"),
			port.Map(1769, newznab.AudioMP3, "|- Ska-Punk, Ska-Core (lossy)"),
			port.Map(1765, newznab.AudioMP3, "|- Reggae (lossy)"),
			port.Map(1771, newznab.AudioMP3, "|- Dub (lossy)"),
			port.Map(1770, newznab.AudioMP3, "|- Dancehall, Raggamuffin (lossy)"),
			port.Map(1768, newznab.AudioLossless, "|- Reggae, Dancehall, Dub (lossless)"),
			port.Map(1774, newznab.AudioLossless, "|- Ska, Ska-Punk, Ska-Jazz (lossless)"),
			port.Map(1772, newznab.Audio, "|- Отечественный Reggae, Dub (lossy и lossless)"),
			port.Map(2233, newznab.Audio, "|- Reggae, Ska, Dub (компиляции) (lossy и lossless)"),
			port.Map(416, newznab.Audio, "Саундтреки, караоке и мюзиклы"),
			port.Map(2377, newznab.AudioVideo, "|- Караоке"),
			port.Map(468, newznab.Audio, "|- Минусовки (lossy и lossless)"),
			port.Map(691, newznab.AudioLossless, "|- Саундтреки к отечественным фильмам (lossless)"),
			port.Map(469, newznab.AudioMP3, "|- Саундтреки к отечественным фильмам (lossy)"),
			port.Map(786, newznab.AudioLossless, "|- Саундтреки к зарубежным фильмам (lossless)"),
			port.Map(785, newznab.AudioMP3, "|- Саундтреки к зарубежным фильмам (lossy)"),
			port.Map(1631, newznab.AudioLossless, "|- Саундтреки к сериалам (lossless)"),
			port.Map(1499, newznab.AudioMP3, "|- Саундтреки к сериалам (lossy)"),
			port.Map(715, newznab.Audio, "|- Саундтреки к мультфильмам (lossy и lossless)"),
			port.Map(1388, newznab.AudioLossless, "|- Саундтреки к аниме (lossless)"),
			port.Map(282, newznab.AudioMP3, "|- Саундтреки к аниме (lossy)"),
			port.Map(796, newznab.AudioMP3, "|- Неофициальные саундтреки к фильмам и сериалам (lossy)"),
			port.Map(784, newznab.AudioLossless, "|- Саундтреки к играм (lossless)"),
			port.Map(783, newznab.AudioMP3, "|- Саундтреки к играм (lossy)"),
			port.Map(2331, newznab.AudioMP3, "|- Неофициальные саундтреки к играм (lossy)"),
			port.Map(2431, newznab.Audio, "|- Аранжировки музыки из игр (lossy и lossless)"),
			port.Map(880, newznab.Audio, "|- Мюзикл (lossy и lossless)"),
			port.Map(1215, newznab.Audio, "Шансон, Авторская и Военная песня"),
			port.Map(1220, newznab.AudioLossless, "|- Отечественный шансон (lossless)"),
			port.Map(1221, newznab.AudioMP3, "|- Отечественный шансон (lossy)"),
			port.Map(1334, newznab.AudioMP3, "|- Сборники отечественного шансона (lossy)"),
			port.Map(1216, newznab.AudioLossless, "|- Военная песня, марши (lossless)"),
			port.Map(1223, newznab.AudioMP3, "|- Военная песня, марши (lossy)"),
			port.Map(1224, newznab.AudioLossless, "|- Авторская песня (lossless)"),
			port.Map(1225, newznab.AudioMP3, "|- Авторская песня (lossy)"),
			port.Map(1226, newznab.Audio, "|- Менестрели и ролевики (lossy и lossless)"),
			port.Map(782, newznab.Audio, "Лейбл- и сцен-паки. Неофициальные сборники и ремастеринги. AI-музыка"),
			port.Map(577, newznab.Audio, "|- AI-Music - музыка ИИ, нейросетей (lossy и lossless)"),
			port.Map(1842, newznab.AudioLossless, "Label Packs (lossless)"),
			port.Map(1648, newznab.AudioMP3, "Label packs, Scene packs (lossy)"),
			port.Map(134, newznab.AudioLossless, "|- Неофициальные сборники и ремастеринги (lossless)"),
			port.Map(965, newznab.AudioMP3, "|- Неофициальные сборники (lossy)"),
			port.Map(2495, newznab.AudioMP3, "Отечественная поп-музыка "),
			port.Map(424, newznab.AudioMP3, "|- Популярная музыка России и стран бывшего СССР (lossy)"),
			port.Map(1361, newznab.AudioMP3, "|- Популярная музыка России и стран бывшего СССР (сборники) (lossy)"),
			port.Map(425, newznab.AudioLossless, "|- Популярная музыка России и стран бывшего СССР (lossless)"),
			port.Map(1635, newznab.AudioMP3, "|- Советская эстрада, ретро, романсы (lossy)"),
			port.Map(1634, newznab.AudioLossless, "|- Советская эстрада, ретро, романсы (lossless)"),
			port.Map(2497, newznab.Audio, "Зарубежная поп-музыка"),
			port.Map(428, newznab.AudioMP3, "|- Зарубежная поп-музыка (lossy)"),
			port.Map(1362, newznab.AudioMP3, "|- Зарубежная поп-музыка (сборники) (lossy)"),
			port.Map(429, newznab.AudioLossless, "|- Зарубежная поп-музыка (lossless)"),
			port.Map(735, newznab.AudioMP3, "|- Итальянская поп-музыка (lossy)"),
			port.Map(1753, newznab.AudioLossless, "|- Итальянская поп-музыка (lossless)"),
			port.Map(2232, newznab.AudioMP3, "|- Латиноамериканская поп-музыка (lossy)"),
			port.Map(714, newznab.AudioLossless, "|- Латиноамериканская поп-музыка (lossless)"),
			port.Map(1331, newznab.AudioMP3, "|- Восточноазиатская поп-музыка (lossy)"),
			port.Map(1330, newznab.AudioLossless, "|- Восточноазиатская поп-музыка (lossless)"),
			port.Map(1219, newznab.AudioMP3, "|- Зарубежный шансон (lossy)"),
			port.Map(1452, newznab.AudioLossless, "|- Зарубежный шансон (lossless)"),
			port.Map(2275, newznab.AudioMP3, "|- Easy Listening, Instrumental Pop (lossy)"),
			port.Map(2270, newznab.AudioLossless, "|- Easy Listening, Instrumental Pop (lossless)"),
			port.Map(1351, newznab.Audio, "|- Сборники песен для детей (lossy и lossless)"),
			port.Map(2499, newznab.Audio, "Eurodance, Disco, Hi-NRG"),
			port.Map(2503, newznab.AudioMP3, "|- Eurodance, Euro-House, Technopop (lossy)"),
			port.Map(2504, newznab.AudioMP3, "|- Eurodance, Euro-House, Technopop (сборники) (lossy)"),
			port.Map(2502, newznab.AudioLossless, "|- Eurodance, Euro-House, Technopop (lossless)"),
			port.Map(2501, newznab.AudioMP3, "|- Disco, Italo-Disco, Euro-Disco, Hi-NRG (lossy)"),
			port.Map(2505, newznab.AudioMP3, "|- Disco, Italo-Disco, Euro-Disco, Hi-NRG (сборники) (lossy)"),
			port.Map(2500, newznab.AudioLossless, "|- Disco, Italo-Disco, Euro-Disco, Hi-NRG (lossless)"),
			port.Map(2267, newznab.Audio, "Зарубежный джаз"),
			port.Map(2277, newznab.AudioLossless, "|- Early Jazz, Swing, Gypsy (lossless)"),
			port.Map(2278, newznab.AudioLossless, "|- Bop (lossless)"),
			port.Map(2279, newznab.AudioLossless, "|- Mainstream Jazz, Cool (lossless)"),
			port.Map(2280, newznab.AudioLossless, "|- Jazz Fusion (lossless)"),
			port.Map(2281, newznab.AudioLossless, "|- World Fusion, Ethnic Jazz (lossless)"),
			port.Map(2282, newznab.AudioLossless, "|- Avant-Garde Jazz, Free Improvisation (lossless)"),
			port.Map(2353, newznab.AudioLossless, "|- Modern Creative, Third Stream (lossless)"),
			port.Map(2284, newznab.AudioLossless, "|- Smooth, Jazz-Pop (lossless)"),
			port.Map(2285, newznab.AudioLossless, "|- Vocal Jazz (lossless)"),
			port.Map(2283, newznab.AudioLossless, "|- Funk, Soul, R&B (lossless)"),
			port.Map(2286, newznab.AudioLossless, "|- Сборники зарубежного джаза (lossless)"),
			port.Map(2287, newznab.AudioMP3, "|- Зарубежный джаз (lossy)"),
			port.Map(2268, newznab.Audio, "Зарубежный блюз"),
			port.Map(2293, newznab.AudioLossless, "|- Blues (Texas, Chicago, Modern and Others) (lossless)"),
			port.Map(2292, newznab.AudioLossless, "|- Blues-rock (lossless)"),
			port.Map(2290, newznab.AudioLossless, "|- Roots, Pre-War Blues, Early R&B, Gospel (lossless)"),
			port.Map(2289, newznab.AudioLossless, "|- Зарубежный блюз (сборники; Tribute VA) (lossless)"),
			port.Map(2288, newznab.AudioMP3, "|- Зарубежный блюз (lossy)"),
			port.Map(2269, newznab.Audio, "Отечественный джаз и блюз"),
			port.Map(2297, newznab.AudioLossless, "|- Отечественный джаз (lossless)"),
			port.Map(2295, newznab.AudioMP3, "|- Отечественный джаз (lossy)"),
			port.Map(2296, newznab.AudioLossless, "|- Отечественный блюз (lossless)"),
			port.Map(2298, newznab.AudioMP3, "|- Отечественный блюз (lossy)"),
			port.Map(1698, newznab.Audio, "Зарубежный Rock"),
			port.Map(1702, newznab.AudioLossless, "|- Classic Rock & Hard Rock (lossless)"),
			port.Map(1703, newznab.AudioMP3, "|- Classic Rock & Hard Rock (lossy)"),
			port.Map(1704, newznab.AudioLossless, "|- Progressive & Art-Rock (lossless)"),
			port.Map(1705, newznab.AudioMP3, "|- Progressive & Art-Rock (lossy)"),
			port.Map(1706, newznab.AudioLossless, "|- Folk-Rock (lossless)"),
			port.Map(1707, newznab.AudioMP3, "|- Folk-Rock (lossy)"),
			port.Map(2329, newznab.AudioLossless, "|- AOR (Melodic Hard Rock, Arena rock) (lossless)"),
			port.Map(2330, newznab.AudioMP3, "|- AOR (Melodic Hard Rock, Arena rock) (lossy)"),
			port.Map(1708, newznab.AudioLossless, "|- Pop-Rock & Soft Rock (lossless)"),
			port.Map(1709, newznab.AudioMP3, "|- Pop-Rock & Soft Rock (lossy)"),
			port.Map(1710, newznab.AudioLossless, "|- Instrumental Guitar Rock (lossless)"),
			port.Map(1711, newznab.AudioMP3, "|- Instrumental Guitar Rock (lossy)"),
			port.Map(1712, newznab.AudioLossless, "|- Rockabilly, Psychobilly, Rock'n'Roll (lossless)"),
			port.Map(1713, newznab.AudioMP3, "|- Rockabilly, Psychobilly, Rock'n'Roll (lossy)"),
			port.Map(731, newznab.AudioLossless, "|- Сборники зарубежного рока (lossless)"),
			port.Map(1799, newznab.AudioMP3, "|- Сборники зарубежного рока (lossy)"),
			port.Map(1714, newznab.AudioLossless, "|- Восточноазиатский рок (lossless)"),
			port.Map(1715, newznab.AudioMP3, "|- Восточноазиатский рок (lossy)"),
			port.Map(1716, newznab.Audio, "Зарубежный Metal"),
			port.Map(1796, newznab.AudioLossless, "|- Avant-garde, Experimental Metal (lossless)"),
			port.Map(1797, newznab.AudioMP3, "|- Avant-garde, Experimental Metal (lossy)"),
			port.Map(1719, newznab.AudioLossless, "|- Black (lossless)"),
			port.Map(1778, newznab.AudioMP3, "|- Black (lossy)"),
			port.Map(1779, newznab.AudioLossless, "|- Death, Doom (lossless)"),
			port.Map(1780, newznab.AudioMP3, "|- Death, Doom (lossy)"),
			port.Map(1720, newznab.AudioLossless, "|- Folk, Pagan, Viking (lossless)"),
			port.Map(798, newznab.AudioMP3, "|- Folk, Pagan, Viking (lossy)"),
			port.Map(1724, newznab.AudioLossless, "|- Gothic Metal (lossless)"),
			port.Map(1725, newznab.AudioMP3, "|- Gothic Metal (lossy)"),
			port.Map(1730, newznab.AudioLossless, "|- Grind, Brutal Death (lossless)"),
			port.Map(1731, newznab.AudioMP3, "|- Grind, Brutal Death (lossy)"),
			port.Map(1726, newznab.AudioLossless, "|- Heavy, Power, Progressive (lossless)"),
			port.Map(1727, newznab.AudioMP3, "|- Heavy, Power, Progressive (lossy)"),
			port.Map(1815, newznab.AudioLossless, "|- Sludge, Stoner, Post-Metal (lossless)"),
			port.Map(1816, newznab.AudioMP3, "|- Sludge, Stoner, Post-Metal (lossy)"),
			port.Map(1728, newznab.AudioLossless, "|- Thrash, Speed (lossless)"),
			port.Map(1729, newznab.AudioMP3, "|- Thrash, Speed (lossy)"),
			port.Map(2230, newznab.AudioLossless, "|- Сборники (lossless)"),
			port.Map(2231, newznab.AudioMP3, "|- Сборники (lossy)"),
			port.Map(1732, newznab.Audio, "Зарубежные Alternative, Punk, Independent"),
			port.Map(1736, newznab.AudioLossless, "|- Alternative & Nu-metal (lossless)"),
			port.Map(1737, newznab.AudioMP3, "|- Alternative & Nu-metal (lossy)"),
			port.Map(1738, newznab.AudioLossless, "|- Punk (lossless)"),
			port.Map(1739, newznab.AudioMP3, "|- Punk (lossy)"),
			port.Map(1740, newznab.AudioLossless, "|- Hardcore (lossless)"),
			port.Map(1741, newznab.AudioMP3, "|- Hardcore (lossy)"),
			port.Map(1773, newznab.AudioLossless, "|- Indie Rock, Indie Pop, Dream Pop, Brit-Pop (lossless)"),
			port.Map(202, newznab.AudioMP3, "|- Indie Rock, Indie Pop, Dream Pop, Brit-Pop (lossy)"),
			port.Map(172, newznab.AudioLossless, "|- Post-Punk, Shoegaze, Garage Rock, Noise Rock (lossless)"),
			port.Map(236, newznab.AudioMP3, "|- Post-Punk, Shoegaze, Garage Rock, Noise Rock (lossy)"),
			port.Map(1742, newznab.AudioLossless, "|- Post-Rock (lossless)"),
			port.Map(1743, newznab.AudioMP3, "|- Post-Rock (lossy)"),
			port.Map(1744, newznab.AudioLossless, "|- Industrial & Post-industrial (lossless)"),
			port.Map(1745, newznab.AudioMP3, "|- Industrial & Post-industrial (lossy)"),
			port.Map(1746, newznab.AudioLossless, "|- Emocore, Post-hardcore, Metalcore (lossless)"),
			port.Map(1747, newznab.AudioMP3, "|- Emocore, Post-hardcore, Metalcore (lossy)"),
			port.Map(1748, newznab.AudioLossless, "|- Gothic Rock & Dark Folk (lossless)"),
			port.Map(1749, newznab.AudioMP3, "|- Gothic Rock & Dark Folk (lossy)"),
			port.Map(2175, newznab.AudioLossless, "|- Avant-garde, Experimental Rock (lossless)"),
			port.Map(2174, newznab.AudioMP3, "|- Avant-garde, Experimental Rock (lossy)"),
			port.Map(722, newznab.Audio, "Отечественный Rock, Metal"),
			port.Map(737, newznab.AudioLossless, "|- Rock (lossless)"),
			port.Map(738, newznab.AudioMP3, "|- Rock (lossy)"),
			port.Map(464, newznab.AudioLossless, "|- Alternative, Punk, Independent (lossless)"),
			port.Map(463, newznab.AudioMP3, "|- Alternative, Punk, Independent (lossy)"),
			port.Map(739, newznab.AudioLossless, "|- Metal (lossless)"),
			port.Map(740, newznab.AudioMP3, "|- Metal (lossy)"),
			port.Map(951, newznab.AudioLossless, "|- Rock на языках народов xUSSR (lossless)"),
			port.Map(952, newznab.AudioMP3, "|- Rock на языках народов xUSSR (lossy)"),
			port.Map(1821, newznab.Audio, "Trance, Goa Trance, Psy-Trance, PsyChill, Ambient, Dub"),
			port.Map(1844, newznab.AudioLossless, "|- Goa Trance, Psy-Trance (lossless)"),
			port.Map(1822, newznab.AudioMP3, "|- Goa Trance, Psy-Trance (lossy)"),
			port.Map(1894, newznab.AudioLossless, "|- PsyChill, Ambient, Dub (lossless)"),
			port.Map(1895, newznab.AudioMP3, "|- PsyChill, Ambient, Dub (lossy)"),
			port.Map(460, newznab.AudioMP3, "|- Goa Trance, Psy-Trance, PsyChill, Ambient, Dub (Live Sets, Mixes) (lossy)"),
			port.Map(1818, newznab.AudioLossless, "|- Trance (lossless)"),
			port.Map(1819, newznab.AudioMP3, "|- Trance (lossy)"),
			port.Map(1847, newznab.AudioMP3, "|- Trance (Singles, EPs) (lossy)"),
			port.Map(1824, newznab.AudioMP3, "|- Trance (Radioshows, Podcasts, Live Sets, Mixes) (lossy)"),
			port.Map(1807, newznab.Audio, "House, Techno, Hardcore, Hardstyle, Jumpstyle"),
			port.Map(1829, newznab.AudioLossless, "|- Hardcore, Hardstyle, Jumpstyle (lossless)"),
			port.Map(1830, newznab.AudioMP3, "|- Hardcore, Hardstyle, Jumpstyle (lossy)"),
			port.Map(1831, newznab.AudioMP3, "|- Hardcore, Hardstyle, Jumpstyle (vinyl, web)"),
			port.Map(1857, newznab.AudioLossless, "|- House (lossless)"),
			port.Map(1859, newznab.AudioMP3, "|- House (Radioshow, Podcast, Liveset, Mixes)"),
			port.Map(1858, newznab.AudioMP3, "|- House (lossy)"),
			port.Map(840, newznab.AudioMP3, "|- House (Проморелизы, сборники) (lossy)"),
			port.Map(1860, newznab.AudioMP3, "|- House (Singles, EPs) (lossy)"),
			port.Map(1825, newznab.AudioLossless, "|- Techno (lossless)"),
			port.Map(1826, newznab.AudioMP3, "|- Techno (lossy)"),
			port.Map(1827, newznab.AudioMP3, "|- Techno (Radioshows, Podcasts, Livesets, Mixes)"),
			port.Map(1828, newznab.AudioMP3, "|- Techno (Singles, EPs) (lossy)"),
			port.Map(1808, newznab.Audio, "Drum & Bass, Jungle, Breakbeat, Dubstep, IDM, Electro"),
			port.Map(797, newznab.AudioLossless, "|- Electro, Electro-Freestyle, Nu Electro (lossless)"),
			port.Map(1805, newznab.AudioMP3, "|- Electro, Electro-Freestyle, Nu Electro (lossy)"),
			port.Map(1832, newznab.AudioLossless, "|- Drum & Bass, Jungle (lossless)"),
			port.Map(1833, newznab.AudioMP3, "|- Drum & Bass, Jungle (lossy)"),
			port.Map(1834, newznab.AudioMP3, "|- Drum & Bass, Jungle (Radioshows, Podcasts, Livesets, Mixes)"),
			port.Map(1836, newznab.AudioLossless, "|- Breakbeat (lossless)"),
			port.Map(1837, newznab.AudioMP3, "|- Breakbeat (lossy)"),
			port.Map(1839, newznab.AudioLossless, "|- Dubstep (lossless)"),
			port.Map(454, newznab.AudioMP3, "|- Dubstep (lossy)"),
			port.Map(1838, newznab.AudioMP3, "|- Breakbeat, Dubstep (Radioshows, Podcasts, Livesets, Mixes)"),
			port.Map(1840, newznab.AudioLossless, "|- IDM (lossless)"),
			port.Map(1841, newznab.AudioMP3, "|- IDM (lossy)"),
			port.Map(2229, newznab.AudioMP3, "|- IDM Discography & Collections (lossy)"),
			port.Map(1809, newznab.Audio, "Chillout, Lounge, Downtempo, Trip-Hop"),
			port.Map(1861, newznab.AudioLossless, "|- Chillout, Lounge, Downtempo (lossless)"),
			port.Map(1862, newznab.AudioMP3, "|- Chillout, Lounge, Downtempo (lossy)"),
			port.Map(1947, newznab.AudioLossless, "|- Nu Jazz, Acid Jazz, Future Jazz (lossless)"),
			port.Map(1946, newznab.AudioMP3, "|- Nu Jazz, Acid Jazz, Future Jazz (lossy)"),
			port.Map(1945, newznab.AudioLossless, "|- Trip Hop, Abstract Hip-Hop (lossless)"),
			port.Map(1944, newznab.AudioMP3, "|- Trip Hop, Abstract Hip-Hop (lossy)"),
			port.Map(1810, newznab.Audio, "Traditional Electronic, Ambient, Modern Classical, Electroacoustic, Experimental"),
			port.Map(1864, newznab.AudioLossless, "|- Traditional Electronic, Ambient (lossless)"),
			port.Map(1865, newznab.AudioMP3, "|- Traditional Electronic, Ambient (lossy)"),
			port.Map(1871, newznab.AudioLossless, "|- Modern Classical, Electroacoustic (lossless)"),
			port.Map(1867, newznab.AudioMP3, "|- Modern Classical, Electroacoustic (lossy)"),
			port.Map(1869, newznab.AudioLossless, "|- Experimental (lossless)"),
			port.Map(1873, newznab.AudioMP3, "|- Experimental (lossy)"),
			port.Map(1811, newznab.Audio, "Industrial, Noise, EBM, Dark Electro, Aggrotech, Cyberpunk, Synthpop, New Wave"),
			port.Map(1868, newznab.AudioLossless, "|- EBM, Dark Electro, Aggrotech (lossless)"),
			port.Map(1875, newznab.AudioMP3, "|- EBM, Dark Electro, Aggrotech (lossy)"),
			port.Map(1877, newznab.AudioLossless, "|- Industrial, Noise (lossless)"),
			port.Map(1878, newznab.AudioMP3, "|- Industrial, Noise (lossy)"),
			port.Map(1907, newznab.Audio, "|- Cyberpunk, 8-bit, Chiptune (lossy & lossless)"),
			port.Map(1880, newznab.AudioLossless, "|- Synthpop, Futurepop, New Wave, Electropop (lossless)"),
			port.Map(1881, newznab.AudioMP3, "|- Synthpop, Futurepop, New Wave, Electropop (lossy)"),
			port.Map(466, newznab.AudioLossless, "|- Synthwave, Spacesynth, Dreamwave, Retrowave, Outrun (lossless)"),
			port.Map(465, newznab.AudioMP3, "|- Synthwave, Spacesynth, Dreamwave, Retrowave, Outrun (lossy)"),
			port.Map(1866, newznab.AudioLossless, "|- Darkwave, Neoclassical, Ethereal, Dungeon Synth (lossless)"),
			port.Map(406, newznab.AudioMP3, "|- Darkwave, Neoclassical, Ethereal, Dungeon Synth (lossy)"),
			port.Map(1299, newznab.Audio, "Hi-Res stereo и многоканальная музыка"),
			port.Map(1884, newznab.Audio, "|- Классика и классика в современной обработке (Hi-Res stereo)"),
			port.Map(1164, newznab.Audio, "|- Классика и классика в современной обработке (многоканальная музыка)"),
			port.Map(2513, newznab.Audio, "|- New Age, Relax, Meditative & Flamenco (Hi-Res stereo и многоканальная музыка)"),
			port.Map(1397, newznab.Audio, "|- Саундтреки (Hi-Res stereo и многоканальная музыка)"),
			port.Map(2512, newznab.Audio, "|- Музыка разных жанров (Hi-Res stereo и многоканальная музыка)"),
			port.Map(1885, newznab.Audio, "|- Поп-музыка (Hi-Res stereo)"),
			port.Map(1163, newznab.Audio, "|- Поп-музыка (многоканальная музыка)"),
			port.Map(2302, newznab.Audio, "|- Джаз и Блюз (Hi-Res stereo)"),
			port.Map(2303, newznab.Audio, "|- Джаз и Блюз (многоканальная музыка)"),
			port.Map(1755, newznab.Audio, "|- Рок-музыка (Hi-Res stereo)"),
			port.Map(1757, newznab.Audio, "|- Рок-музыка (многоканальная музыка)"),
			port.Map(1893, newznab.Audio, "|- Электронная музыка (Hi-Res stereo)"),
			port.Map(1890, newznab.Audio, "|- Электронная музыка (многоканальная музыка)"),
			port.Map(2219, newznab.Audio, "Оцифровки с аналоговых носителей"),
			port.Map(1660, newznab.Audio, "|- Классика и классика в современной обработке (оцифровки)"),
			port.Map(506, newznab.Audio, "|- Фольклор, народная и этническая музыка (оцифровки)"),
			port.Map(1835, newznab.Audio, "|- Rap, Hip-Hop, R'n'B, Reggae, Ska, Dub (оцифровки)"),
			port.Map(1625, newznab.Audio, "|- Саундтреки и мюзиклы (оцифровки)"),
			port.Map(1217, newznab.Audio, "|- Шансон, авторские, военные песни и марши (оцифровки)"),
			port.Map(974, newznab.Audio, "|- Музыка других жанров (оцифровки)"),
			port.Map(1444, newznab.Audio, "|- Зарубежная поп-музыка (оцифровки)"),
			port.Map(2401, newznab.Audio, "|- Советская эстрада, ретро, романсы (оцифровки)"),
			port.Map(239, newznab.Audio, "|- Отечественная поп-музыка (оцифровки)"),
			port.Map(450, newznab.Audio, "|- Инструментальная поп-музыка (оцифровки)"),
			port.Map(2301, newznab.Audio, "|- Джаз и блюз (оцифровки)"),
			port.Map(123, newznab.Audio, "|- Alternative, Punk, Independent (оцифровки)"),
			port.Map(1756, newznab.Audio, "|- Зарубежная рок-музыка (оцифровки)"),
			port.Map(1758, newznab.Audio, "|- Отечественная рок-музыка (оцифровки)"),
			port.Map(1766, newznab.Audio, "|- Зарубежный и Отечественный Metal (оцифровки)"),
			port.Map(1754, newznab.Audio, "|- Электронная музыка (оцифровки)"),
			port.Map(860, newznab.Audio, "Неофициальные конверсии цифровых форматов"),
			port.Map(453, newznab.Audio, "|- Конверсии Quadraphonic"),
			port.Map(1170, newznab.Audio, "|- Конверсии SACD"),
			port.Map(1759, newznab.Audio, "|- Конверсии Blu-Ray, ADVD и DVD-Audio"),
			port.Map(1852, newznab.Audio, "|- Апмиксы-Upmixes/Даунмиксы-Downmix"),
			port.Map(413, newznab.AudioVideo, "Музыкальное SD видео"),
			port.Map(445, newznab.AudioVideo, "|- Классическая и современная академическая музыка (Видео)"),
			port.Map(702, newznab.AudioVideo, "|- Опера, Оперетта и Мюзикл (Видео)"),
			port.Map(1990, newznab.AudioVideo, "|- Балет и современная хореография (Видео)"),
			port.Map(1793, newznab.AudioVideo, "|- Классика в современной обработке, Classical Crossover (Видео)"),
			port.Map(1141, newznab.AudioVideo, "|- Фольклор, Народная и Этническая музыка и фламенко (Видео)"),
			port.Map(1775, newznab.AudioVideo, "|- New Age, Relax, Meditative, Рэп, Хип-Хоп, R'n'B, Reggae, Ska, Dub (Видео)"),
			port.Map(1227, newznab.AudioVideo, "|- Зарубежный и Отечественный Шансон, Авторская и Военная песня (Видео)"),
			port.Map(475, newznab.AudioVideo, "|- Музыка других жанров, Советская эстрада, ретро, романсы (Видео)"),
			port.Map(1121, newznab.AudioVideo, "|- Отечественная поп-музыка (Видео)"),
			port.Map(431, newznab.AudioVideo, "|- Зарубежная Поп-музыка, Eurodance, Disco (Видео)"),
			port.Map(2378, newznab.AudioVideo, "|- Восточноазиатская поп-музыка (Видео)"),
			port.Map(2383, newznab.AudioVideo, "|- Разножанровые сборные концерты и сборники видеоклипов (Видео)"),
			port.Map(2305, newznab.AudioVideo, "|- Джаз и Блюз (Видео)"),
			port.Map(1782, newznab.AudioVideo, "|- Rock (Видео)"),
			port.Map(1787, newznab.AudioVideo, "|- Metal (Видео)"),
			port.Map(1789, newznab.AudioVideo, "|- Зарубежный Alternative, Punk, Independent (Видео)"),
			port.Map(1791, newznab.AudioVideo, "|- Отечественный Рок, Панк, Альтернатива (Видео)"),
			port.Map(1912, newznab.AudioVideo, "|- Электронная музыка (Видео)"),
			port.Map(1189, newznab.AudioVideo, "|- Документальные фильмы о музыке и музыкантах (Видео)"),
			port.Map(2403, newznab.AudioVideo, "Музыкальное DVD видео"),
			port.Map(984, newznab.AudioVideo, "|- Классическая и современная академическая музыка (DVD Видео)"),
			port.Map(983, newznab.AudioVideo, "|- Опера, Оперетта и Мюзикл (DVD Видео)"),
			port.Map(2352, newznab.AudioVideo, "|- Балет и современная хореография (DVD Видео)"),
			port.Map(2384, newznab.AudioVideo, "|- Классика в современной обработке, Classical Crossover (DVD Видео)"),
			port.Map(1142, newznab.AudioVideo, "|- Фольклор, Народная и Этническая музыка и Flamenco (DVD Видео)"),
			port.Map(1107, newznab.AudioVideo, "|- New Age, Relax, Meditative, Рэп, Хип-Хоп, R'n'B, Reggae, Ska, Dub (DVD Видео)"),
			port.Map(1228, newznab.AudioVideo, "|- Зарубежный и Отечественный Шансон, Авторская и Военная песня (DVD Видео)"),
			port.Map(988, newznab.AudioVideo, "|- Музыка других жанров, Советская эстрада, ретро, романсы (DVD Видео)"),
			port.Map(1122, newznab.AudioVideo, "|- Отечественная поп-музыка (DVD Видео)"),
			port.Map(986, newznab.AudioVideo, "|- Зарубежная Поп-музыка, Eurodance, Disco (DVD Видео)"),
			port.Map(2379, newznab.AudioVideo, "|- Восточноазиатская поп-музыка (DVD Видео)"),
			port.Map(2088, newznab.AudioVideo, "|- Разножанровые сборные концерты и сборники видеоклипов (DVD Видео)"),
			port.Map(2304, newznab.AudioVideo, "|- Джаз и Блюз (DVD Видео)"),
			port.Map(1783, newznab.AudioVideo, "|- Зарубежный Rock (DVD Видео)"),
			port.Map(1788, newznab.AudioVideo, "|- Зарубежный Metal (DVD Видео)"),
			port.Map(1790, newznab.AudioVideo, "|- Зарубежный Alternative, Punk, Independent (DVD Видео)"),
			port.Map(1792, newznab.AudioVideo, "|- Отечественный Рок, Метал, Панк, Альтернатива (DVD Видео)"),
			port.Map(1886, newznab.AudioVideo, "|- Электронная музыка (DVD Видео)"),
			port.Map(2509, newznab.AudioVideo, "|- Документальные фильмы о музыке и музыкантах (DVD Видео)"),
			port.Map(2507, newznab.AudioVideo, "Неофициальные DVD видео"),
			port.Map(2263, newznab.AudioVideo, "|- Классическая музыка, Опера, Балет, Мюзикл (Неофициальные DVD Видео)"),
			port.Map(2511, newznab.AudioVideo, "|- Шансон, Авторская песня, Сборные концерты, МДЖ (Неофициальные DVD Видео)"),
			port.Map(2264, newznab.AudioVideo, "|- Зарубежная и Отечественная Поп-музыка (Неофициальные DVD Видео)"),
			port.Map(2262, newznab.AudioVideo, "|- Джаз и Блюз (Неофициальные DVD Видео)"),
			port.Map(2261, newznab.AudioVideo, "|- Зарубежная и Отечественная Рок-музыка (Неофициальные DVD Видео)"),
			port.Map(1887, newznab.AudioVideo, "|- Электронная музыка (Неофициальные DVD Видео)"),
			port.Map(2531, newznab.AudioVideo, "|- Прочие жанры (Неофициальные DVD Видео)"),
			port.Map(2400, newznab.AudioVideo, "Музыкальное HD видео"),
			port.Map(1812, newznab.AudioVideo, "|- Классическая и современная академическая музыка (HD Видео)"),
			port.Map(655, newznab.AudioVideo, "|- Опера, Оперетта и Мюзикл (HD Видео)"),
			port.Map(1777, newznab.AudioVideo, "|- Балет и современная хореография (HD Видео)"),
			port.Map(2530, newznab.AudioVideo, "|- Фольклор, Народная, Этническая музыка и Flamenco (HD Видео)"),
			port.Map(2529, newznab.AudioVideo, "|- New Age, Relax, Meditative, Рэп, Хип-Хоп, R'n'B, Reggae, Ska, Dub (HD Видео)"),
			port.Map(1781, newznab.AudioVideo, "|- Музыка других жанров, Разножанровые сборные концерты (HD видео)"),
			port.Map(2508, newznab.AudioVideo, "|- Зарубежная поп-музыка (HD Видео)"),
			port.Map(2426, newznab.AudioVideo, "|- Отечественная поп-музыка (HD видео)"),
			port.Map(2351, newznab.AudioVideo, "|- Восточноазиатская Поп-музыка (HD Видео)"),
			port.Map(2306, newznab.AudioVideo, "|- Джаз и Блюз (HD Видео)"),
			port.Map(1795, newznab.AudioVideo, "|- Зарубежный рок (HD Видео)"),
			port.Map(2271, newznab.AudioVideo, "|- Отечественный рок (HD видео)"),
			port.Map(1913, newznab.AudioVideo, "|- Электронная музыка (HD Видео)"),
			port.Map(1784, newznab.AudioVideo, "|- UHD музыкальное видео"),
			port.Map(1892, newznab.AudioVideo, "|- Документальные фильмы о музыке и музыкантах (HD Видео)"),
			port.Map(2266, newznab.AudioVideo, "|- Официальные апскейлы (Blu-ray, HDTV, WEB-DL)"),
			port.Map(518, newznab.AudioVideo, "Некондиционное музыкальное видео (Видео, DVD видео, HD видео)"),
			port.Map(5, newznab.PCGames, "Игры для Windows"),
			port.Map(635, newznab.PCGames, "|- Горячие Новинки"),
			port.Map(127, newznab.PCGames, "|- Аркады"),
			port.Map(2203, newznab.PCGames, "|- Файтинги"),
			port.Map(647, newznab.PCGames, "|- Экшены от первого лица"),
			port.Map(646, newznab.PCGames, "|- Экшены от третьего лица"),
			port.Map(50, newznab.PCGames, "|- Хорроры"),
			port.Map(53, newznab.PCGames, "|- Приключения и квесты"),
			port.Map(1008, newznab.PCGames, "|- Квесты в стиле \"Поиск предметов\""),
			port.Map(900, newznab.PCGames, "|- Визуальные новеллы"),
			port.Map(128, newznab.PCGames, "|- Для самых маленьких"),
			port.Map(2204, newznab.PCGames, "|- Логические игры"),
			port.Map(278, newznab.PCGames, "|- Шахматы"),
			port.Map(52, newznab.PCGames, "|- Ролевые игры"),
			port.Map(54, newznab.PCGames, "|- Симуляторы"),
			port.Map(51, newznab.PCGames, "|- Стратегии в реальном времени"),
			port.Map(2226, newznab.PCGames, "|- Пошаговые стратегии"),
			port.Map(2118, newznab.PCGames, "|- Антологии и сборники игр"),
			port.Map(1310, newznab.PCGames, "|- Старые игры (Экшены)"),
			port.Map(2410, newznab.PCGames, "|- Старые игры (Ролевые игры)"),
			port.Map(2205, newznab.PCGames, "|- Старые игры (Стратегии)"),
			port.Map(2225, newznab.PCGames, "|- Старые игры (Приключения и квесты)"),
			port.Map(2206, newznab.PCGames, "|- Старые игры (Симуляторы)"),
			port.Map(2228, newznab.PCGames, "|- IBM-PC-несовместимые компьютеры"),
			port.Map(139, newznab.PCGames, "Прочее для Windows-игр"),
			port.Map(2478, newznab.PCGames, "|- Официальные патчи, моды, плагины, дополнения"),
			port.Map(2480, newznab.PCGames, "|- Неофициальные модификации, плагины, дополнения"),
			port.Map(2481, newznab.PCGames, "|- Русификаторы"),
			port.Map(2142, newznab.PCGames, "Прочее для Microsoft Flight Simulator, Prepar3D, X-Plane"),
			port.Map(2060, newznab.PCGames, "|- Сценарии, меши и аэропорты для FS2004, FSX, P3D"),
			port.Map(2145, newznab.PCGames, "|- Самолёты и вертолёты для FS2004, FSX, P3D"),
			port.Map(2146, newznab.PCGames, "|- Миссии, трафик, звуки, паки и утилиты для FS2004, FSX, P3D"),
			port.Map(2143, newznab.PCGames, "|- Сценарии, миссии, трафик, звуки, паки и утилиты для X-Plane"),
			port.Map(2012, newznab.PCGames, "|- Самолёты и вертолёты для X-Plane"),
			port.Map(960, newznab.PCMac, "Игры для Apple Macintosh"),
			port.Map(537, newznab.PCMac, "|- Нативные игры для Mac"),
			port.Map(637, newznab.PCMac, "|- Игры для Mac с Wineskin, DOSBox, Cider и другими"),
			port.Map(899, newznab.PCGames, "Игры для Linux"),
			port.Map(1992, newznab.PCGames, "|- Нативные игры для Linux"),
			port.Map(2059, newznab.PCGames, "|- Игры для Linux с Wine, DOSBox и другими"),
			port.Map(548, newznab.Console, "Игры для консолей"),
			port.Map(908, newznab.Console, "|- PS"),
			port.Map(357, newznab.ConsoleOther, "|- PS2"),
			port.Map(886, newznab.ConsolePS3, "|- PS3"),
			port.Map(973, newznab.ConsolePS4, "|- PS4"),
			port.Map(546, newznab.ConsoleOther, "|- PS5"),
			port.Map(1352, newznab.ConsolePSP, "|- PSP"),
			port.Map(1116, newznab.ConsolePSP, "|- Игры PS1 для PSP"),
			port.Map(595, newznab.ConsolePSVita, "|- PS Vita"),
			port.Map(887, newznab.ConsoleXBox, "|- Original Xbox"),
			port.Map(510, newznab.ConsoleXBox360, "|- Xbox 360"),
			port.Map(773, newznab.ConsoleWii, "|- Wii/WiiU"),
			port.Map(774, newznab.ConsoleNDS, "|- NDS/3DS"),
			port.Map(1605, newznab.Console, "|- Switch"),
			port.Map(968, newznab.Console, "|- Dreamcast"),
			port.Map(129, newznab.Console, "|- Остальные платформы"),
			port.Map(2185, newznab.ConsoleOther, "Видео для консолей"),
			port.Map(2487, newznab.ConsoleOther, "|- Видео для PS Vita"),
			port.Map(2182, newznab.ConsoleOther, "|- Фильмы для PSP"),
			port.Map(2181, newznab.ConsoleOther, "|- Сериалы для PSP"),
			port.Map(2180, newznab.ConsoleOther, "|- Мультфильмы для PSP"),
			port.Map(2179, newznab.ConsoleOther, "|- Дорамы для PSP"),
			port.Map(2186, newznab.ConsoleOther, "|- Аниме для PSP"),
			port.Map(700, newznab.ConsoleOther, "|- Видео для PSP"),
			port.Map(1926, newznab.ConsoleOther, "|- Видео для PS3 и других консолей"),
			port.Map(650, newznab.PCMobileOther, "Игры для мобильных устройств"),
			port.Map(2149, newznab.PCMobileAndroid, "|- Игры для Android"),
			port.Map(2420, newznab.ConsoleOther, "|- Игры для Oculus Quest"),
			port.Map(1004, newznab.PCMobileOther, "|- Игры для Symbian"),
			port.Map(1002, newznab.PCMobileOther, "|- Игры для Windows Mobile"),
			port.Map(240, newznab.OtherMisc, "Игровое видео"),
			port.Map(2415, newznab.OtherMisc, "|- Видеопрохождения игр"),
			port.Map(1012, newznab.PC, "Операционные системы от Microsoft"),
			port.Map(2489, newznab.PC, "|- Оригинальные образы Windows"),
			port.Map(2523, newznab.PC, "|- Сборки Windows 8 и далее"),
			port.Map(2153, newznab.PC, "|- Сборки Windows XP - Windows 7"),
			port.Map(1019, newznab.PC, "|- Операционные системы выпущенные до Windows XP"),
			port.Map(1021, newznab.PC, "|- Серверные ОС (оригинальные + сборки)"),
			port.Map(1025, newznab.PC, "|- Разное (сборки All-in-One, пакеты обновлений, утилиты, и прочее)"),
			port.Map(1376, newznab.PC, "Linux, Unix и другие ОС"),
			port.Map(1379, newznab.PC, "|- Операционные системы (Linux, Unix)"),
			port.Map(1381, newznab.PC, "|- Программное обеспечение (Linux, Unix)"),
			port.Map(1473, newznab.PC, "|- Другие ОС и ПО под них"),
			port.Map(1013, newznab.PC, "Системные программы"),
			port.Map(1028, newznab.PC, "|- Работа с жёстким диском"),
			port.Map(1029, newznab.PC, "|- Резервное копирование"),
			port.Map(1030, newznab.PC, "|- Архиваторы и файловые менеджеры"),
			port.Map(1031, newznab.PC, "|- Программы для настройки и оптимизации ОС"),
			port.Map(1032, newznab.PC, "|- Сервисное обслуживание компьютера"),
			port.Map(1033, newznab.PC, "|- Работа с носителями информации"),
			port.Map(1034, newznab.PC, "|- Информация и диагностика"),
			port.Map(1066, newznab.PC, "|- Программы для интернет и сетей"),
			port.Map(1035, newznab.PC, "|- ПО для защиты компьютера (Антивирусное ПО, Фаерволлы)"),
			port.Map(1536, newznab.PC, "|- Драйверы и прошивки"),
			port.Map(1051, newznab.PC, "|- Оригинальные диски к компьютерам и комплектующим"),
			port.Map(1040, newznab.PC, "|- Серверное ПО для Windows"),
			port.Map(1041, newznab.PC, "|- Изменение интерфейса ОС Windows"),
			port.Map(1636, newznab.PC, "|- Скринсейверы"),
			port.Map(1042, newznab.PC, "|- Разное (Системные программы под Windows)"),
			port.Map(1014, newznab.PC, "Системы для бизнеса, офиса, научной и проектной работы"),
			port.Map(2134, newznab.PC, "|- Медицина - интерактивный софт"),
			port.Map(1060, newznab.PC, "|- Всё для дома: кройка, шитьё, кулинария"),
			port.Map(1061, newznab.PC, "|- Офисные системы"),
			port.Map(1062, newznab.PC, "|- Системы для бизнеса"),
			port.Map(1067, newznab.PC, "|- Распознавание текста, звука и синтез речи"),
			port.Map(1086, newznab.PC, "|- Работа с PDF и DjVu"),
			port.Map(1068, newznab.PC, "|- Словари, переводчики"),
			port.Map(1063, newznab.PC, "|- Системы для научной работы"),
			port.Map(1087, newznab.PC, "|- САПР (общие и машиностроительные)"),
			port.Map(1192, newznab.PC, "|- САПР (электроника, автоматика, ГАП)"),
			port.Map(1088, newznab.PC, "|- Программы для архитекторов и строителей"),
			port.Map(1193, newznab.PC, "|- Библиотеки и проекты для архитекторов и дизайнеров интерьеров"),
			port.Map(1071, newznab.PC, "|- Прочие справочные системы"),
			port.Map(1073, newznab.PC, "|- Разное (Системы для бизнеса, офиса, научной и проектной работы)"),
			port.Map(1052, newznab.PC, "Веб-разработка и Программирование"),
			port.Map(1053, newznab.PC, "|- WYSIWYG Редакторы для веб-диза"),
			port.Map(1054, newznab.PC, "|- Текстовые редакторы с подсветкой"),
			port.Map(1055, newznab.PC, "|- Среды программирования, компиляторы и вспомогательные программы"),
			port.Map(1056, newznab.PC, "|- Компоненты для сред программирования"),
			port.Map(2077, newznab.PC, "|- Системы управления базами данных"),
			port.Map(1057, newznab.PC, "|- Скрипты и движки сайтов, CMS а также расширения к ним"),
			port.Map(1018, newznab.PC, "|- Шаблоны для сайтов и CMS"),
			port.Map(1058, newznab.PC, "|- Разное (Веб-разработка и программирование)"),
			port.Map(1016, newznab.PC, "Программы для работы с мультимедиа и 3D"),
			port.Map(1195, newznab.PC, "|- Тестовые диски для настройки аудио/видео аппаратуры"),
			port.Map(1079, newznab.PC, "|- Программные комплекты"),
			port.Map(1080, newznab.PC, "|- Плагины для программ компании Adobe"),
			port.Map(1081, newznab.PC, "|- Графические редакторы"),
			port.Map(1082, newznab.PC, "|- Программы для верстки, печати и работы со шрифтами"),
			port.Map(1083, newznab.PC, "|- 3D моделирование, рендеринг и плагины для них"),
			port.Map(1084, newznab.PC, "|- Анимация"),
			port.Map(1085, newznab.PC, "|- Создание BD/HD/DVD-видео"),
			port.Map(1089, newznab.PC, "|- Редакторы видео"),
			port.Map(1090, newznab.PC, "|- Видео- Аудио- конверторы"),
			port.Map(1065, newznab.PC, "|- Аудио- и видео-, CD- проигрыватели и каталогизаторы"),
			port.Map(1064, newznab.PC, "|- Каталогизаторы и просмотрщики графики"),
			port.Map(1092, newznab.PC, "|- Разное (Программы для работы с мультимедиа и 3D)"),
			port.Map(1204, newznab.PC, "|- Виртуальные студии, секвенсоры и аудиоредакторы"),
			port.Map(1027, newznab.PC, "|- Виртуальные инструменты и синтезаторы"),
			port.Map(1199, newznab.PC, "|- Плагины для обработки звука"),
			port.Map(1091, newznab.PC, "|- Разное (Программы для работы со звуком)"),
			port.Map(828, newznab.OtherMisc, "Материалы для мультимедиа и дизайна"),
			port.Map(1357, newznab.OtherMisc, "|- Авторские работы"),
			port.Map(890, newznab.OtherMisc, "|- Официальные сборники векторных клипартов"),
			port.Map(830, newznab.OtherMisc, "|- Прочие векторные клипарты"),
			port.Map(1290, newznab.OtherMisc, "|- Photostocks"),
			port.Map(1962, newznab.OtherMisc, "|- Дополнения для программ компоузинга и постобработки"),
			port.Map(831, newznab.OtherMisc, "|- Рамки, шаблоны, текстуры и фоны"),
			port.Map(829, newznab.OtherMisc, "|- Прочие растровые клипарты"),
			port.Map(633, newznab.OtherMisc, "|- 3D модели, сцены и материалы"),
			port.Map(1009, newznab.OtherMisc, "|- Футажи"),
			port.Map(1963, newznab.OtherMisc, "|- Прочие сборники футажей"),
			port.Map(1954, newznab.OtherMisc, "|- Музыкальные библиотеки"),
			port.Map(1010, newznab.OtherMisc, "|- Звуковые эффекты"),
			port.Map(1674, newznab.OtherMisc, "|- Библиотеки сэмплов"),
			port.Map(2421, newznab.OtherMisc, "|- Библиотеки и саундбанки для сэмплеров, пресеты для синтезаторов"),
			port.Map(2492, newznab.OtherMisc, "|- Multitracks"),
			port.Map(839, newznab.OtherMisc, "|- Материалы для создания меню и обложек DVD"),
			port.Map(1679, newznab.OtherMisc, "|- Дополнения, стили, кисти, формы, узоры для программ Adobe"),
			port.Map(1011, newznab.OtherMisc, "|- Шрифты"),
			port.Map(835, newznab.OtherMisc, "|- Разное (Материалы для мультимедиа и дизайна)"),
			port.Map(1503, newznab.OtherMisc, "ГИС, системы навигации и карты"),
			port.Map(1507, newznab.OtherMisc, "|- ГИС (Геоинформационные системы)"),
			port.Map(1526, newznab.OtherMisc, "|- Карты, снабженные программной оболочкой"),
			port.Map(1508, newznab.OtherMisc, "|- Атласы и карты современные (после 1950 г.)"),
			port.Map(1509, newznab.OtherMisc, "|- Атласы и карты старинные (до 1950 г.)"),
			port.Map(1510, newznab.OtherMisc, "|- Карты прочие (астрономические, исторические, тематические)"),
			port.Map(1511, newznab.OtherMisc, "|- Встроенная автомобильная навигация"),
			port.Map(1512, newznab.OtherMisc, "|- Garmin"),
			port.Map(1513, newznab.OtherMisc, "|- Ozi"),
			port.Map(1514, newznab.OtherMisc, "|- TomTom"),
			port.Map(1515, newznab.OtherMisc, "|- Navigon / Navitel"),
			port.Map(1516, newznab.OtherMisc, "|- Igo"),
			port.Map(1517, newznab.OtherMisc, "|- Разное - системы навигации и карты"),
			port.Map(285, newznab.PCMobileOther, "Приложения для мобильных устройств"),
			port.Map(2154, newznab.PCMobileAndroid, "|- Приложения для Android"),
			port.Map(1005, newznab.PCMobileOther, "|- Приложения для Java"),
			port.Map(289, newznab.PCMobileOther, "|- Приложения для Symbian"),
			port.Map(290, newznab.PCMobileOther, "|- Приложения для Windows Mobile"),
			port.Map(288, newznab.PCMobileOther, "|- Софт для работы с телефоном"),
			port.Map(292, newznab.PCMobileOther, "|- Прошивки для телефонов"),
			port.Map(291, newznab.PCMobileOther, "|- Обои и темы"),
			port.Map(957, newznab.PCMobileOther, "Видео для мобильных устройств"),
			port.Map(287, newznab.PCMobileOther, "|- Видео для смартфонов и КПК"),
			port.Map(286, newznab.PCMobileOther, "|- Видео в формате 3GP для мобильных"),
			port.Map(1366, newznab.PCMac, "Apple Macintosh"),
			port.Map(1368, newznab.PCMac, "|- Mac OS (для Macintosh)"),
			port.Map(1383, newznab.PCMac, "|- Mac OS (для РС-Хакинтош)"),
			port.Map(1394, newznab.PCMac, "|- Программы для просмотра и обработки видео (Mac OS)"),
			port.Map(1370, newznab.PCMac, "|- Программы для создания и обработки графики (Mac OS)"),
			port.Map(2237, newznab.PCMac, "|- Плагины для программ компании Adobe (Mac OS)"),
			port.Map(1372, newznab.PCMac, "|- Аудио редакторы и конвертеры (Mac OS)"),
			port.Map(1373, newznab.PCMac, "|- Системные программы (Mac OS)"),
			port.Map(1375, newznab.PCMac, "|- Офисные программы (Mac OS)"),
			port.Map(1371, newznab.PCMac, "|- Программы для интернета и сетей (Mac OS)"),
			port.Map(1374, newznab.PCMac, "|- Другие программы (Mac OS)"),
			port.Map(1933, newznab.PCMobileiOS, "iOS"),
			port.Map(1935, newznab.PCMobileiOS, "|- Программы для iOS"),
			port.Map(1003, newznab.PCMobileiOS, "|- Игры для iOS"),
			port.Map(1937, newznab.PCMobileiOS, "|- Разное для iOS"),
			port.Map(2235, newznab.PCMobileiOS, "Видео"),
			port.Map(1908, newznab.PCMobileiOS, "|- Фильмы для iPod, iPhone, iPad"),
			port.Map(864, newznab.PCMobileiOS, "|- Сериалы для iPod, iPhone, iPad"),
			port.Map(863, newznab.PCMobileiOS, "|- Мультфильмы для iPod, iPhone, iPad"),
			port.Map(2535, newznab.PCMobileiOS, "|- Аниме для iPod, iPhone, iPad"),
			port.Map(2534, newznab.PCMobileiOS, "|- Музыкальное видео для iPod, iPhone, iPad"),
			port.Map(2238, newznab.PCMac, "Видео HD"),
			port.Map(1936, newznab.PCMac, "|- Фильмы HD для Apple TV"),
			port.Map(315, newznab.PCMac, "|- Сериалы HD для Apple TV"),
			port.Map(1363, newznab.PCMac, "|- Мультфильмы HD для Apple TV"),
			port.Map(2082, newznab.PCMac, "|- Документальное видео HD для Apple TV"),
			port.Map(2241, newznab.PCMac, "|- Музыкальное видео HD для Apple TV"),
			port.Map(2236, newznab.Audio, "Аудио"),
			port.Map(1909, newznab.AudioAudiobook, "|- Аудиокниги (AAC, ALAC)"),
			port.Map(1927, newznab.AudioLossless, "|- Музыка lossless (ALAC)"),
			port.Map(2240, newznab.Audio, "|- Музыка Lossy (AAC-iTunes)"),
			port.Map(2248, newznab.Audio, "|- Музыка Lossy (AAC)"),
			port.Map(2244, newznab.Audio, "|- Музыка Lossy (AAC) (Singles, EPs)"),
			port.Map(10, newznab.OtherMisc, "Разное (раздачи)"),
			port.Map(865, newznab.OtherMisc, "|- Психоактивные аудиопрограммы"),
			port.Map(1100, newznab.OtherMisc, "|- Аватары, Иконки, Смайлы"),
			port.Map(1643, newznab.OtherMisc, "|- Живопись, Графика, Скульптура, Digital Art"),
			port.Map(848, newznab.OtherMisc, "|- Картинки"),
			port.Map(808, newznab.OtherMisc, "|- Любительские фотографии"),
			port.Map(630, newznab.OtherMisc, "|- Обои"),
			port.Map(1664, newznab.OtherMisc, "|- Фото знаменитостей"),
			port.Map(148, newznab.Audio, "|- Аудио"),
			port.Map(807, newznab.TVOther, "|- Видео"),
			port.Map(147, newznab.Books, "|- Публикации и учебные материалы (тексты)"),
			port.Map(847, newznab.MoviesOther, "|- Трейлеры и дополнительные материалы к фильмам"),
			port.Map(1167, newznab.TVOther, "|- Любительские видеоклипы"),
			port.Map(321, newznab.Other, "|- Отчеты о встречах"),
		),
	}
}

func GetSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return GetPageableRequests(ix, searchCriteria.SearchTerm, searchCriteria.Categories, 0), nil
}

func GetTvSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	season := 0
	if searchCriteria.Season != nil {
		season = *searchCriteria.Season
	}
	return GetPageableRequests(ix, searchCriteria.SearchTerm, searchCriteria.Categories, season), nil
}

func GetPageableRequests(ix *ix, searchTerm string, categories []newznab.ID, season int) port.Chain {
	var pageableRequests port.Chain

	if len(categories) > 0 {
		trackerCategories := ix.Categories.TrackerIDs(categories)

		// RuTracker supports maximum 200 categories in one search
		for trackerCategoriesChunk := range slices.Chunk(trackerCategories, 200) {
			pageableRequests.Add(GetPagedRequests(ix, searchTerm, trackerCategoriesChunk, season))
		}
	} else {
		pageableRequests.Add(GetPagedRequests(ix, searchTerm, nil, season))
	}

	return pageableRequests
}

var searchStringWildcards = regexp.MustCompile("[^a-zA-Zа-яА-ЯёЁ0-9]+")

func GetPagedRequests(ix *ix, term string, trackerCategories []string, season int) *port.Request {
	var parameters stringutil.NameValueCollection

	searchString := term

	// if the search string is empty use the getnew view
	if strings.TrimSpace(searchString) == "" {
		parameters.Set("nm", searchString)
	} else {
		//  replace any space, special char, etc. with % (wildcard)
		searchString = searchStringWildcards.ReplaceAllString(searchString, "%")
		searchString = strings.ReplaceAll(searchString, "-", " ")

		if season != 0 {
			searchString += " ТВ | Сезон: " + strconv.Itoa(season)
		}

		parameters.Set("nm", searchString)
	}

	if len(trackerCategories) > 0 {
		parameters.Set("f", strings.Join(trackerCategories, ","))
	}

	searchURL := ix.URL("forum/tracker.php")

	if parameters.Count() > 0 {
		searchURL.RawQuery = parameters.GetQueryString()
	}

	return port.Get(searchURL).Accept("text/html").NoRedirect()
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll("table#tor-tbl > tbody > tr")

	for _, row := range rows {
		release, err := ParseReleaseRow(ix, row)
		if err != nil {
			return nil, err
		}
		if release != nil {
			releaseInfos = append(releaseInfos, *release)
		}
	}

	sort.SliceStable(releaseInfos, func(i, j int) bool { return releaseInfos[i].PublishDate.After(releaseInfos[j].PublishDate) })

	return releaseInfos, nil
}

func ParseReleaseRow(ix *ix, row *dom.Node) (*port.TorrentInfo, error) {
	qDownloadLink := row.QuerySelector("td.tor-size > a.tr-dl")

	// Expects moderation
	if qDownloadLink == nil {
		return nil, nil //nolint:nilnil // a row awaiting moderation is no Release and no error, as the C#'s null
	}

	qDetailsLink := row.QuerySelector("td.t-title-col > div.t-title > a.tLink")
	infoURL := ix.BaseURL.String() + "forum/" + qDetailsLink.GetAttribute("href")
	downloadURL := ix.BaseURL.String() + "forum/" + qDownloadLink.GetAttribute("href")

	title := strings.TrimSpace(qDetailsLink.TextContent())
	categories, err := GetCategoryOfRelease(ix, row)
	if err != nil {
		return nil, err
	}

	size, err := GetSizeOfRelease(row)
	if err != nil {
		return nil, err
	}

	seeders, err := GetSeedersOfRelease(row)
	if err != nil {
		return nil, err
	}
	leechers, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(8)").TextContent())
	if err != nil {
		return nil, err
	}

	grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(9)").TextContent())
	if err != nil {
		return nil, err
	}

	publishDate, err := GetPublishDateOfRelease(ix, row)
	if err != nil {
		return nil, err
	}

	release := &port.TorrentInfo{
		GUID:        infoURL,
		InfoURL:     infoURL,
		DownloadURL: downloadURL,
		Title:       titleParser.Parse(title, categories, ix.Settings.RussianLetters, ix.Settings.MoveFirstTagsToEndOfReleaseTitle, ix.Settings.MoveAllTagsToEndOfReleaseTitle, ix.Settings.AddRussianToTitle),
		// SlipStream: a Release has no description, so the site's own
		// title isn't kept beside the parsed one.
		Categories:           categories,
		Size:                 size,
		Seeders:              seeders,
		Peers:                leechers + seeders,
		Grabs:                grabs,
		PublishDate:          publishDate,
		DownloadVolumeFactor: 1,
		UploadVolumeFactor:   1,
		MinimumRatio:         1,
		MinimumSeedTime:      0,
	}
	if ix.Settings.UseMagnetLinks {
		release.DownloadURL = infoURL
	}

	return release, nil
}

func GetSeedersOfRelease(row *dom.Node) (int, error) {
	seeders := 0
	qSeeders := row.QuerySelector("td:nth-child(7)")
	if qSeeders != nil && !strings.Contains(qSeeders.TextContent(), "дн") {
		seedersString := qSeeders.QuerySelector("b").TextContent()
		if strings.TrimSpace(seedersString) != "" {
			var err error
			if seeders, err = parseutil.CoerceInt(seedersString); err != nil {
				return 0, err
			}
		}
	}

	return seeders, nil
}

func GetCategoryOfRelease(ix *ix, row *dom.Node) ([]newznab.Category, error) {
	forum := row.QuerySelector("td.f-name-col > div.f-name > a").GetAttribute("href")
	cat, err := parseutil.GetArgumentFromQueryString(forum, "f")
	if err != nil {
		return nil, err
	}

	return ix.Categories.FromTracker(cat), nil
}

func GetSizeOfRelease(row *dom.Node) (int64, error) {
	return parseutil.GetBytes(row.QuerySelector("td.tor-size").GetAttribute("data-ts_text"))
}

func GetPublishDateOfRelease(ix *ix, row *dom.Node) (time.Time, error) {
	timestamp := row.QuerySelector("td:nth-child(10)").GetAttribute("data-ts_text")
	secs, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("read the upload time %q: %w", timestamp, err)
	}
	return datetimeutil.UnixTimestampToDateTime(secs, ix.Clock), nil
}

// RuTrackerTitleParser is RuTracker's title cleanup. Its patterns are
// .NET's, so they run on regexp2, which reads them as .NET does: balancing
// groups, and \w, \W, \s, \d and \b over all of Unicode.
// SlipStream: regexp2 has no \p{IsCyrillic} block, so it is its range,
// \u0400-\u04FF.
type RuTrackerTitleParser struct{}

var titleParser RuTrackerTitleParser

func dotnet(pattern string, options regexp2.RegexOptions) *regexp2.Regexp {
	return regexp2.MustCompile(pattern, options)
}

// replace is Regex.Replace. regexp2 errs only on a MatchTimeout, which none
// of these sets, or on a replacement it can't read, which a test would meet.
func replace(re *regexp2.Regexp, input, replacement string) string {
	out, err := re.Replace(input, replacement, -1, -1)
	if err != nil {
		panic(err)
	}
	return out
}

func isMatch(re *regexp2.Regexp, input string) bool {
	ok, err := re.MatchString(input)
	if err != nil {
		panic(err)
	}
	return ok
}

// matches is Regex.Matches: each match's index, in runes, and value.
func matches(re *regexp2.Regexp, input string) []*regexp2.Match {
	var out []*regexp2.Match
	m, err := re.FindStringMatch(input)
	for ; m != nil && err == nil; m, err = re.FindNextMatch(m) {
		out = append(out, m)
	}
	if err != nil {
		panic(err)
	}
	return out
}

var (
	FindTagsInTitlesRegexList = []*regexp2.Regexp{
		dotnet(`\((?>\((?<c>)|[^()]+|\)(?<-c>))*(?(c)(?!))\)`, regexp2.None),
		dotnet(`\[(?>\[(?<c>)|[^\[\]]+|\](?<-c>))*(?(c)(?!))\]`, regexp2.None),
	}

	stripCyrillicRegex = dotnet(`(\([\u0400-\u04FF\W]+\))|(^[\u0400-\u04FF\W\d]+\/ )|([\u0400-\u04FF \-]+,+)|([\u0400-\u04FF]+)`, regexp2.IgnoreCase)

	tvTitleCommaRegex     = dotnet(`\s(\d+),(\d+)`, regexp2.None)
	tvTitleCyrillicXRegex = dotnet(`([\s-])Х+([\s\)\]])`, regexp2.IgnoreCase)

	tvTitleRusSeasonEpisodeOfRegex = dotnet(`Сезон\s*[:]*\s+(\d+).+(?:Серии|Эпизод|Выпуски)+\s*[:]*\s+(\d+(?:-\d+)?)\s*из\s*([\w?])`, regexp2.IgnoreCase)
	tvTitleRusSeasonEpisodeRegex   = dotnet(`Сезон\s*[:]*\s+(\d+).+(?:Серии|Эпизод|Выпуски)+\s*[:]*\s+(\d+(?:-\d+)?)`, regexp2.IgnoreCase)
	tvTitleRusSeasonRegex          = dotnet(`Сезон\s*[:]*\s+(\d+(?:-\d+)?)`, regexp2.IgnoreCase)
	tvTitleRusEpisodeOfRegex       = dotnet(`(?:Серии|Эпизод|Выпуски)+\s*[:]*\s+(\d+(?:-\d+)?)\s*из\s*([\w?])`, regexp2.IgnoreCase)
	tvTitleRusEpisodeRegex         = dotnet(`(?:Серии|Эпизод|Выпуски)+\s*[:]*\s+(\d+(?:-\d+)?)`, regexp2.IgnoreCase)
	tvTitleRusSeasonAnimeRegex     = dotnet(`ТВ[-]*(?:(\d+))`, regexp2.IgnoreCase)
	tvTitleRusEpisodeAnimeOfRegex  = dotnet(`\[(\d+(\+\d+)?)\s+из\s+(\d+(\+\d+)?)\]`, regexp2.IgnoreCase)

	dashRegex       = dotnet(`\p{Pd}`, regexp2.IgnoreCase)
	double4KRegex   = dotnet(`\b(2160p), 4K\b`, regexp2.IgnoreCase)
	blurayDiscRegex = dotnet(`\bBlu-ray Disc\b`, regexp2.IgnoreCase)
	directorRegex   = dotnet(`(\([\u0400-\u04FF\W]+)(?:\s/\s(.+?))?\)`, regexp2.IgnoreCase)
	voRegex         = dotnet(`((?:\dx\s)?(?:[A-Z])?VO\s\(.+?\))`, regexp2.None)
	r5Regex         = dotnet(`(.*)(.R5.)(.*)`, regexp2.None)
	subRegex        = dotnet(`(\bSub\b[^+]*\b|\b[\+]*Sub[\+]*\b)`, regexp2.None)
	// The C# writes "\bRUS\b" in a regular string, where \b is a backspace,
	// so this matches no title and RUS is always added. Go's "\b" is the
	// same backspace.
	rusRegex           = dotnet("\bRUS\b", regexp2.IgnoreCase)
	ripRegex           = dotnet(`\b-Rip\b`, regexp2.IgnoreCase)
	hdtvRipRegex       = dotnet(`\bHDTVRip\b`, regexp2.IgnoreCase)
	webDLRipRegex      = dotnet(`\bWEB-DLRip\b`, regexp2.IgnoreCase)
	webDLRipShortRegex = dotnet(`\bWEBDLRip\b`, regexp2.IgnoreCase)
	webDLRegex         = dotnet(`\bWEBDL\b`, regexp2.IgnoreCase)
	kurazhRegex        = dotnet(`\bКураж-Бамбей\b`, regexp2.IgnoreCase)
	openSlashRegex     = dotnet(`\(\s*\/\s*`, regexp2.None)
	closeSlashRegex    = dotnet(`\s*\/\s*\)`, regexp2.None)
	emptyTagRegex      = dotnet(`[\[\(]\s*[\)\]]`, regexp2.None)
	plusesRegex        = dotnet(`\s+\+(?:\s+\+)+\s+`, regexp2.None)
	spacesRegex        = dotnet(`\s+`, regexp2.None)
)

func (RuTrackerTitleParser) Parse(title string, categories []newznab.Category, stripCyrillicLetters, moveFirstTagsToEndOfReleaseTitle, moveAllTagsToEndOfReleaseTitle, addRussianToTitle bool) string {
	// https://www.fileformat.info/info/unicode/category/Pd/list.htm
	title = replace(dashRegex, title, "-")

	// replace double 4K quality in title
	title = replace(double4KRegex, title, "$1")

	if IsAnyTvCategory(categories) {
		title = replace(tvTitleCommaRegex, title, " $1-$2")
		title = replace(tvTitleCyrillicXRegex, title, "$1XX$2")

		title = replace(tvTitleRusSeasonEpisodeOfRegex, title, "S$1E$2 of $3")
		title = replace(tvTitleRusSeasonEpisodeRegex, title, "S$1E$2")
		title = replace(tvTitleRusSeasonRegex, title, "S$1")
		title = replace(tvTitleRusEpisodeOfRegex, title, "E$1 of $2")
		title = replace(tvTitleRusEpisodeRegex, title, "E$1")
		title = replace(tvTitleRusSeasonAnimeRegex, title, "S$1")
		title = replace(tvTitleRusEpisodeAnimeOfRegex, title, "E$1 of $3")
	} else if IsAnyMovieCategory(categories) {
		// Bluray quality fix: radarr parse Blu-ray Disc as Bluray-1080p but should be BR-DISK
		title = replace(blurayDiscRegex, title, "BR-DISK")
	}

	if IsAnyTvCategory(categories) || IsAnyMovieCategory(categories) {
		// remove director's name from title
		// rutracker movies titles look like: russian name / english name (russian director / english director) other stuff
		// Ирландец / The Irishman (Мартин Скорсезе / Martin Scorsese) [2019, США, криминал, драма, биография, WEB-DL 1080p] Dub (Пифагор) + MVO (Jaskier) + AVO (Юрий Сербин) + Sub Rus, Eng + Original Eng
		// this part should be removed: (Мартин Скорсезе / Martin Scorsese)
		title = replace(directorRegex, title, "")

		// Remove VO, MVO and DVO from titles
		title = replace(voRegex, title, "")

		// Remove R5 and (R5) from release names
		title = replace(r5Regex, title, "$1")

		// Remove Sub languages from release names
		title = replace(subRegex, title, "")
	}

	// language fix: all rutracker releases contains russian track
	if addRussianToTitle && (IsAnyTvCategory(categories) || IsAnyMovieCategory(categories)) && !isMatch(rusRegex, title) {
		title += " RUS"
	}

	if stripCyrillicLetters {
		title = strings.Trim(replace(stripCyrillicRegex, title, ""), " -")
	}

	if moveAllTagsToEndOfReleaseTitle {
		title = MoveAllTagsToEndOfReleaseTitle(title)
	} else if moveFirstTagsToEndOfReleaseTitle {
		title = MoveFirstTagsToEndOfReleaseTitle(title)
	}

	title = replace(ripRegex, title, "Rip")
	title = replace(hdtvRipRegex, title, "HDTV")
	title = replace(webDLRipRegex, title, "WEB-DL")
	title = replace(webDLRipShortRegex, title, "WEB-DL")
	title = replace(webDLRegex, title, "WEB-DL")
	title = replace(kurazhRegex, title, "kurazh")

	title = replace(openSlashRegex, title, "(")
	title = replace(closeSlashRegex, title, ")")

	title = replace(emptyTagRegex, title, "")

	title = replace(plusesRegex, title, " + ")

	title = strings.Trim(title, ` &,.!?+-_|/\:`)

	// replace multiple spaces with a single space
	title = replace(spacesRegex, title, " ")

	return strings.TrimSpace(title)
}

func IsAnyTvCategory(category []newznab.Category) bool {
	return isAny(category, newznab.TV)
}

func IsAnyMovieCategory(category []newznab.Category) bool {
	return isAny(category, newznab.Movies)
}

// isAny is category.Contains(parent) or one of parent.SubCategories.
func isAny(category []newznab.Category, parent newznab.Category) bool {
	return slices.ContainsFunc(category, func(c newznab.Category) bool {
		return c.ID == parent.ID || (c.ID < newznab.CustomBase && c.Parent == parent.ID)
	})
}

func MoveAllTagsToEndOfReleaseTitle(input string) string {
	output := input
	for _, findTagsRegex := range FindTagsInTitlesRegexList {
		for _, match := range matches(findTagsRegex, input) {
			tag := match.String()
			output = strings.TrimSpace(strings.ReplaceAll(output, tag, "") + " " + tag)
		}
	}

	return strings.TrimSpace(output)
}

func MoveFirstTagsToEndOfReleaseTitle(input string) string {
	output := input
	for _, findTagsRegex := range FindTagsInTitlesRegexList {
		expectedIndex := 0
		for _, match := range matches(findTagsRegex, output) {
			if match.Index > expectedIndex {
				substring := string([]rune(output)[expectedIndex:match.Index])
				if strings.TrimSpace(substring) != "" {
					break
				}
				expectedIndex = match.Index
			}

			tag := match.String()
			output = strings.TrimSpace(strings.Replace(output, tag, "", 1) + " " + tag)
			expectedIndex += match.Length
		}
	}

	return strings.TrimSpace(output)
}

type RuTrackerSettings struct {
	port.UserPass
	// SlipStream: labels and help reworded per WRITING.md.
	RussianLetters                   bool `port:"russianLetters" label:"Strip Russian letters" help:"Remove Cyrillic letters from each Release title."`
	UseMagnetLinks                   bool `port:"useMagnetLinks" label:"Use Magnet links" help:"Grab each Release by its Magnet link in place of its .torrent."`
	AddRussianToTitle                bool `port:"addRussianToTitle" label:"Add RUS to titles" help:"Add RUS to the end of each movie and TV Release title, because every RuTracker Release has a Russian track. English-only Releases then read as Russian too."`
	MoveFirstTagsToEndOfReleaseTitle bool `port:"moveFirstTagsToEndOfReleaseTitle" label:"Move leading tags to the end of the title" help:"Move the bracketed tags at the start of each Release title to its end."`
	MoveAllTagsToEndOfReleaseTitle   bool `port:"moveAllTagsToEndOfReleaseTitle" label:"Move all tags to the end of the title" help:"Move every bracketed tag in each Release title to its end, in place of only the leading ones."`
}
