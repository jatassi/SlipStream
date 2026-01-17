package mock

// movieResultsJSON contains pre-loaded search results keyed by TMDB ID.
var movieResultsJSON = map[int]string{
	603: `[
  {
    "guid": "https://seedpool.org/torrent/download/48228.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.1999.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.REMUX-FraMeSToR.mkv",
    "downloadUrl": "https://seedpool.org/torrent/download/48228.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/48228",
    "size": 56808632124,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 133093,
    "tmdbId": 603,
    "quality": "2160p",
    "source": "Remux",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/50766.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.Revolutions.2003.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.REMUX-FraMeSToR.mkv",
    "downloadUrl": "https://seedpool.org/torrent/download/50766.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/50766",
    "size": 54833014305,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 242653,
    "tmdbId": 605,
    "quality": "2160p",
    "source": "Remux",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/50157.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.Reloaded.2003.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.REMUX-FraMeSToR.mkv",
    "downloadUrl": "https://seedpool.org/torrent/download/50157.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/50157",
    "size": 75203194023,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 234215,
    "tmdbId": 604,
    "quality": "2160p",
    "source": "Remux",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/57023.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.Resurrections.2021.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/57023.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/57023",
    "size": 72021857369,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 10838180,
    "tmdbId": 624860,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/143802.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.1999.MULTi.iNTERNAL.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HDR10.HEVC.REMUX-seedpool",
    "downloadUrl": "https://seedpool.org/torrent/download/143802.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/143802",
    "size": 61282183944,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 133093,
    "tmdbId": 603,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/142110.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.1999.2160p.MA.WEB-DL.TrueHD.Atmos.7.1.DV.HDR.H.265-FLUX",
    "downloadUrl": "https://seedpool.org/torrent/download/142110.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/142110",
    "size": 30909468282,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 133093,
    "tmdbId": 603,
    "quality": "2160p",
    "source": "WEB-DL",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/147820.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.Reloaded.2003.MULTi.iNTERNAL.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HDR10.HEVC.REMUX-seedpool",
    "downloadUrl": "https://seedpool.org/torrent/download/147820.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/147820",
    "size": 81204785030,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 234215,
    "tmdbId": 604,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/483180.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.Reloaded.2003.1080p.BluRay.DTS.x264-Geek",
    "downloadUrl": "https://seedpool.org/torrent/download/483180.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/483180",
    "size": 18052468023,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 234215,
    "tmdbId": 604,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/326526.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.Revolutions.2003.Remastered.BluRay.1080p.DDP.5.1.x264-hallowed",
    "downloadUrl": "https://seedpool.org/torrent/download/326526.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/326526",
    "size": 10369502082,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 242653,
    "tmdbId": 605,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/443752.a9cd63925268a6d61face5d10e3ce190",
    "title": "The.Matrix.1999.UHD.BluRay.2160p.DDP.7.1.DV.HDR.x265-hallowed",
    "downloadUrl": "https://seedpool.org/torrent/download/443752.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/443752",
    "size": 17249117073,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 133093,
    "tmdbId": 603,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://mockindexer.org/torrent/download/990001.mock",
    "title": "The.Matrix.1999.2160p.WEB-DL.SDR.AV1.DDP.5.1-MOCK",
    "downloadUrl": "https://mockindexer.org/torrent/download/990001.mock",
    "infoUrl": "https://mockindexer.org/torrents/990001",
    "size": 8589934592,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "mock (API)",
    "protocol": "torrent",
    "imdbId": 133093,
    "tmdbId": 603,
    "quality": "2160p",
    "source": "WEB-DL",
    "resolution": 2160
  },
  {
    "guid": "https://mockindexer.org/torrent/download/990002.mock",
    "title": "The.Matrix.1999.1080p.BluRay.SDR.x264.DTS-HD.MA.5.1-MOCK",
    "downloadUrl": "https://mockindexer.org/torrent/download/990002.mock",
    "infoUrl": "https://mockindexer.org/torrents/990002",
    "size": 12884901888,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "mock (API)",
    "protocol": "torrent",
    "imdbId": 133093,
    "tmdbId": 603,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  }
]`,
	27205: `[
  {
    "guid": "https://seedpool.org/torrent/download/50817.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.UHD.BluRay.2160p.DTS-HD.MA.5.1.DV.HEVC.HYBRID.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/50817.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/50817",
    "size": 71140086723,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/174811.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.2160p.BluRay.DTS-HD.MA.5.1.DV.HDR10.x265-MainFrame",
    "downloadUrl": "https://seedpool.org/torrent/download/174811.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/174811",
    "size": 37933674703,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/418447.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.BluRay.1080p.DDP.5.1.x264-hallowed",
    "downloadUrl": "https://seedpool.org/torrent/download/418447.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/418447",
    "size": 11909900310,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/461060.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.1080p.MA.WEB-DL.DDP5.1.H264-HHWEB",
    "downloadUrl": "https://seedpool.org/torrent/download/461060.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/461060",
    "size": 9002133939,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/647031.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.MULTi.iNTERNAL.UHD.BluRay.2160p.DTS-HD.MA.5.1.HDR10.REMUX-seedpool",
    "downloadUrl": "https://seedpool.org/torrent/download/647031.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/647031",
    "size": 83015057801,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/607643.a9cd63925268a6d61face5d10e3ce190",
    "title": "Cruel.Intentions.1999.BluRay.1080p.DTS-HD.MA.5.1.AVC.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/607643.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/607643",
    "size": 24191582650,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 139134,
    "tmdbId": 796,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/632983.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.UHD.BluRay.2160p.DTS-HD.MA.5.1.HEVC.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/632983.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/632983",
    "size": 70896037991,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/455274.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception.2010.2160p.MA.WEB-DL.DTS-HD.MA.5.1.DV.HDR.H.265-TheFarm",
    "downloadUrl": "https://seedpool.org/torrent/download/455274.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/455274",
    "size": 31518432930,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "WEB-DL",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/282220.a9cd63925268a6d61face5d10e3ce190",
    "title": "Inception 2010 1080p UHD BluRay DD+ 5.1 DV HDR x265-HiDt",
    "downloadUrl": "https://seedpool.org/torrent/download/282220.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/282220",
    "size": 17048151843,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/307219.a9cd63925268a6d61face5d10e3ce190",
    "title": "Cruel Intentions 1999 1080p BluRay DTS 5.1 x264-TDD",
    "downloadUrl": "https://seedpool.org/torrent/download/307219.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/307219",
    "size": 12902883416,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 139134,
    "tmdbId": 796,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://mockindexer.org/torrent/download/990003.mock",
    "title": "Inception.2010.2160p.WEB-DL.SDR.AV1.DDP.5.1-MOCK",
    "downloadUrl": "https://mockindexer.org/torrent/download/990003.mock",
    "infoUrl": "https://mockindexer.org/torrents/990003",
    "size": 9663676416,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "mock (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "2160p",
    "source": "WEB-DL",
    "resolution": 2160
  },
  {
    "guid": "https://mockindexer.org/torrent/download/990004.mock",
    "title": "Inception.2010.1080p.BluRay.SDR.x264.DTS-HD.MA.5.1-MOCK",
    "downloadUrl": "https://mockindexer.org/torrent/download/990004.mock",
    "infoUrl": "https://mockindexer.org/torrents/990004",
    "size": 11811160064,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "mock (API)",
    "protocol": "torrent",
    "imdbId": 1375666,
    "tmdbId": 27205,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  }
]`,
	693134: `[
  {
    "guid": "https://seedpool.org/torrent/download/1815.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.Part.Two.2024.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/1815.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/1815",
    "size": 69026891608,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15239678,
    "tmdbId": 693134,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/34867.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.Part.One.2021.Hybrid.2160p.UHD.BluRay.REMUX.DV.HDR10Plus.HEVC.TrueHD.7.1.Atmos-WiLDCAT",
    "downloadUrl": "https://seedpool.org/torrent/download/34867.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/34867",
    "size": 75973665026,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1160419,
    "tmdbId": 438631,
    "quality": "2160p",
    "source": "Remux",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/48019.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.2021.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/48019.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/48019",
    "size": 74365568150,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1160419,
    "tmdbId": 438631,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/138488.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune AKA Dune: Part One 2021 1080p HMAX WEB-DL DD+ 5.1 Atmos H.264-FLUX",
    "downloadUrl": "https://seedpool.org/torrent/download/138488.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/138488",
    "size": 10499595903,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1160419,
    "tmdbId": 438631,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/175271.a9cd63925268a6d61face5d10e3ce190",
    "title": "Anatomie.d.Une.Chute.AKA.Anatomy.of.a.Fall.2023.1080p.BluRay.DDP5.1.x264-PTer",
    "downloadUrl": "https://seedpool.org/torrent/download/175271.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/175271",
    "size": 19107249747,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 17009710,
    "tmdbId": 915935,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/563.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.Part.Two.2024.REPACK.1080p.AMZN.WEB-DL.DDP5.1.Atmos.H.264-FLUX",
    "downloadUrl": "https://seedpool.org/torrent/download/563.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/563",
    "size": 8558403574,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15239678,
    "tmdbId": 693134,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/121488.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.Prophecy.S01.1080p.MAX.WEB-DL.DDP5.1.Atmos.H.264-FLUX",
    "downloadUrl": "https://seedpool.org/torrent/download/121488.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/121488",
    "size": 7660585699,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      2
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 10466872,
    "tmdbId": 90228,
    "tvdbId": 367118,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/409823.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune (2021) (2160p MA WEB-DL Hybrid H265 DV HDR DDP Atmos 5.1 English - HONE)",
    "downloadUrl": "https://seedpool.org/torrent/download/409823.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/409823",
    "size": 29778002835,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1160419,
    "tmdbId": 438631,
    "quality": "2160p",
    "source": "WEB-DL",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/430555.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.Part.Two.2024.2160p.WEB-DL.DDP5.1.Atmos.DV.HDR.H.265-FLUX",
    "downloadUrl": "https://seedpool.org/torrent/download/430555.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/430555",
    "size": 31418995949,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15239678,
    "tmdbId": 693134,
    "quality": "2160p",
    "source": "WEB-DL",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/315950.a9cd63925268a6d61face5d10e3ce190",
    "title": "Dune.Prophecy.S01.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRID.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/315950.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/315950",
    "size": 214466956936,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      2
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 10466872,
    "tmdbId": 90228,
    "tvdbId": 367118,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  }
]`,
	872585: `[
  {
    "guid": "https://seedpool.org/torrent/download/50995.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.UHD.BluRay.2160p.DTS-HD.MA.5.1.DV.HEVC.HYBRID.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/50995.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/50995",
    "size": 89100999892,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/633269.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.UHD.BluRay.2160p.DTS-HD.MA.5.1.HEVC.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/633269.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/633269",
    "size": 88614629151,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/186045.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.REPACK.BluRay.1080p.DD.5.1.x264-BHDStudio",
    "downloadUrl": "https://seedpool.org/torrent/download/186045.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/186045",
    "size": 11631694628,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/549270.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.1080p.AMZN.WEB-DL.DDP5.1.H.264-GPRS",
    "downloadUrl": "https://seedpool.org/torrent/download/549270.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/549270",
    "size": 11612317395,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/420404.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer (2023) (1080p MA WEB-DL H265 SDR DDP 5.1 English - HONE)",
    "downloadUrl": "https://seedpool.org/torrent/download/420404.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/420404",
    "size": 11513528225,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/421425.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer 2023 Hybrid IMAX 2160p UHD BluRay DTS-HD MA 5.1 DV HDR x265-HiDt",
    "downloadUrl": "https://seedpool.org/torrent/download/421425.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/421425",
    "size": 48228786648,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/346844.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.2160p.PROPER.IMAX.HYBRID.UHD.REMUX.DV.HDR10+.TrueHD.7.1.Atmos-jennaortegaUHD",
    "downloadUrl": "https://seedpool.org/torrent/download/346844.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/346844",
    "size": 126580431072,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "2160p",
    "source": "Remux",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/160652.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.iNTERNAL.UHD.BluRay.2160p.DTS-HD.MA.5.1.HDR.10Bit.x265-BETA",
    "downloadUrl": "https://seedpool.org/torrent/download/160652.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/160652",
    "size": 23912593880,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/218795.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.IMAX.UHD.BluRay.1080p.DD+Atmos.5.1.DoVi.HDR10.x265-SM737",
    "downloadUrl": "https://seedpool.org/torrent/download/218795.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/218795",
    "size": 9296161702,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/658796.a9cd63925268a6d61face5d10e3ce190",
    "title": "Oppenheimer.2023.BluRay.1080p.DTS-HD.MA.5.1.AVC.REMUX",
    "downloadUrl": "https://seedpool.org/torrent/download/658796.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/658796",
    "size": 24761860096,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 15398776,
    "tmdbId": 872585,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  }
]`,
	346698: `[
  {
    "guid": "https://seedpool.org/torrent/download/243463.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie.2023.1080p.MA.WEB-DL.DDP5.1.Atmos.H.264-FLUX",
    "downloadUrl": "https://seedpool.org/torrent/download/243463.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/243463",
    "size": 7432919167,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1517268,
    "tmdbId": 346698,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/57035.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie.2023.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRID.REMUX-FraMeSToR.mkv",
    "downloadUrl": "https://seedpool.org/torrent/download/57035.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/57035",
    "size": 63014610254,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1517268,
    "tmdbId": 346698,
    "quality": "2160p",
    "source": "Remux",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/372829.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie.2023.BluRay.1080p.DDP.5.1.x264-hallowed",
    "downloadUrl": "https://seedpool.org/torrent/download/372829.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/372829",
    "size": 9116717835,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1517268,
    "tmdbId": 346698,
    "quality": "1080p",
    "source": "BluRay",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/565678.a9cd63925268a6d61face5d10e3ce190",
    "title": "Georgie and Mandys First Marriage S02E04 Dirty Hands and a Barbed-Wire Fence 1080p AMZN WEB-DL DDP5 1 H 264-NTb",
    "downloadUrl": "https://seedpool.org/torrent/download/565678.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/565678",
    "size": 1148584598,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      2
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 31589662,
    "tmdbId": 243875,
    "tvdbId": 448023,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/565615.a9cd63925268a6d61face5d10e3ce190",
    "title": "Georgie.Mandy.s.First.Marriage.S02E04.Dirty.Hands.and.a.Barbed-Wire.Fence.1080p.AMZN.WEB-DL.DDP5.1.H.264-BLOOM",
    "downloadUrl": "https://seedpool.org/torrent/download/565615.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/565615",
    "size": 1148587164,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      2
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 31589662,
    "tmdbId": 243875,
    "tvdbId": 448023,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/160760.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie.2023.iNTERNAL.UHD.BluRay.2160p.DTS-HD.MA.7.1.HDR.10Bit.x265-BETA",
    "downloadUrl": "https://seedpool.org/torrent/download/160760.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/160760",
    "size": 13122400184,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1517268,
    "tmdbId": 346698,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/374218.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie.2023.UHD.BluRay.2160p.DDP.7.1.DV.HDR.x265-hallowed",
    "downloadUrl": "https://seedpool.org/torrent/download/374218.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/374218",
    "size": 16140782383,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1517268,
    "tmdbId": 346698,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/331419.a9cd63925268a6d61face5d10e3ce190",
    "title": "Sweeney.Todd.The.Demon.Barber.of.Fleet.Street.2007.UHD.BluRay.2160p.TrueHD.5.1.DV.HEVC.HYBRID.REMUX-FraMeSToR",
    "downloadUrl": "https://seedpool.org/torrent/download/331419.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/331419",
    "size": 62227046012,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 408236,
    "tmdbId": 13885,
    "quality": "2160p",
    "source": "BluRay",
    "resolution": 2160
  },
  {
    "guid": "https://seedpool.org/torrent/download/445077.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie.as.Rapunzel.2002.1080p.OKRU.x264-ICE77",
    "downloadUrl": "https://seedpool.org/torrent/download/445077.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/445077",
    "size": 5310285218,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 313255,
    "tmdbId": 15015,
    "quality": "1080p",
    "resolution": 1080
  },
  {
    "guid": "https://seedpool.org/torrent/download/493606.a9cd63925268a6d61face5d10e3ce190",
    "title": "Barbie (2023) (1080p MA WEB-DL H265 SDR DDP Atmos 5.1 English - HONE)",
    "downloadUrl": "https://seedpool.org/torrent/download/493606.a9cd63925268a6d61face5d10e3ce190",
    "infoUrl": "https://seedpool.org/torrents/493606",
    "size": 7369481518,
    "publishDate": "0001-01-01T00:00:00Z",
    "categories": [
      1
    ],
    "indexerId": 3,
    "indexer": "seedpool (API)",
    "protocol": "torrent",
    "imdbId": 1517268,
    "tmdbId": 346698,
    "quality": "1080p",
    "source": "WEB-DL",
    "resolution": 1080
  }
]`,
}

