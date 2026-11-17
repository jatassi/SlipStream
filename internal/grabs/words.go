package grabs

// Words is what a failure code says to the Admin: why the Grab failed, and
// what to do next. Recovery is empty for a code with nothing to do.
type Words struct {
	Cause    string
	Recovery string
}

const (
	searchAgainWords = "Search again to pick another release."
	nextReleaseWords = "SlipStream grabs the next acceptable release from the same search, if there is one."
)

// failureWords is the one table of failure words: the Activity card, the
// Work page and the Grab failed notification all read it.
var failureWords = map[Code]Words{
	IndexerUnavailable: {"The Indexer didn't respond to the request for the release.", "Search again once the Indexer is back."},
	IndexerRemoved:     {"The Indexer was removed before SlipStream fetched the release's .torrent file.", nextReleaseWords},
	IndexerStopped:     {"The Indexer stopped searching before SlipStream fetched the release's .torrent file.", nextReleaseWords},
	HomeOffline: {
		"The Indexer's home Node was offline for longer than the Offline grace setting allows, and SlipStream never fetched the release's .torrent file.",
		nextReleaseWords,
	},
	TorrentUnreachable: {"SlipStream couldn't download the release's .torrent file or magnet link from the Indexer.", searchAgainWords},
	MagnetInvalid:      {"The release's magnet link has no info hash SlipStream can read.", searchAgainWords},
	NoMetadata:         {"No peer sent the magnet link's metadata in time.", searchAgainWords},
	NoPeerSource: {
		"The magnet link names no tracker, and DHT is off on the Node that downloads it.",
		"Turn on DHT for public torrents in that Node's torrent client settings, or search again.",
	},
	TorrentInvalid:   {"SlipStream can't read the release's .torrent file, or the metadata peers sent for its magnet link.", searchAgainWords},
	InfohashMismatch: {"The .torrent file isn't the release the Indexer listed.", searchAgainWords},
	TorrentV2Only:    {"The release only offers a BitTorrent v2 .torrent, which SlipStream can't download.", searchAgainWords},
	DownloadFailed:   {"The download stopped with an error.", searchAgainWords},
	MarkedFailed:     {"An Admin marked the download failed. SlipStream added the release to the Blocklist.", searchAgainWords},
	Removed:          {"An Admin removed the torrent before SlipStream imported it.", searchAgainWords},
	ClientRemoved:    {"The download client removed the torrent before SlipStream imported it.", searchAgainWords},
	ClientDeleted:    {"An Admin deleted the download client before SlipStream imported the torrent.", searchAgainWords},
	GoalMet:          {"The torrent met its seeding goal and was removed before SlipStream imported it.", searchAgainWords},
	NoVideoFile:      {"The download holds no video file.", searchAgainWords},
	ProbeFailed:      {"SlipStream couldn't read the downloaded video.", searchAgainWords},
	FillsNothing:     {"The downloaded video doesn't meet any Target this Work still wants.", searchAgainWords},
	NameTaken:        {"Another file already has the name this video would take in the Work's folder.", "Move or rename the other file, then search again."},
	NameTooLong:      {"The video's path would be longer than its library root allows.", "Shorten this Content Type's naming templates, then search again."},
	TransferFailed:   {Cause: "The finished download couldn't be copied between Nodes."},
	ImportFailed:     {Cause: "SlipStream couldn't put the file in the library."},
	NodeWrittenOff: {
		"An Admin had SlipStream stop waiting for the Node downloading this Grab. The files it downloaded stay on that Node.",
		searchAgainWords,
	},
	RootChanged: {"The Work moved to another library root after this download was planned.", searchAgainWords},
	Canceled:    {"An Admin canceled this Grab.", searchAgainWords},
	Blocklisted: {"The release's torrent is on the Blocklist.", nextReleaseWords},
	ReleaseFault: {
		"The tracker reported the torrent unregistered, or it never downloaded any data. SlipStream added the release to the Blocklist.",
		searchAgainWords,
	},
	ClientError: {
		`The download client reported an error on the torrent for longer than the setting "A client error fails its Grab after" allows.`,
		nextReleaseWords,
	},
	Declined:   {Cause: "You chose not to import this download."},
	Unreadable: {"ffprobe can't read the downloaded video. SlipStream added the release to the Blocklist.", nextReleaseWords},
}

// WordsOf is code's failure words.
func WordsOf(code Code) Words { return failureWords[code] }
