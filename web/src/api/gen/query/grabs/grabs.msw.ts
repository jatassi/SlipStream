import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  GrabsBlocklist,
  GrabsGrab,
  GrabsGrabList,
  GrabsIndexerGrabList,
  GrabsManualSearch,
  GrabsManualStarted,
  GrabsOfferList,
  GrabsPickResults,
  GrabsReleaseList,
  GrabsSearchList,
  GrabsSearchStarted,
} from "../../model";

export const getListBlocklistResponseMock = (
  overrideResponse: Partial<Extract<GrabsBlocklist, object>> = {},
): GrabsBlocklist => ({
  entries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      because: faker.string.alpha({ length: { min: 10, max: 20 } }),
      cause: faker.helpers.arrayElement([
        "unregistered",
        "no_payload",
        "engine_error",
        "marked_failed",
        "by_hand",
      ] as const),
      indexer: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      infoHash: faker.helpers.fromRegExp("^[0-9a-f]{40}$"),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      work: faker.helpers.arrayElement([
        { id: faker.string.uuid(), title: faker.string.alpha({ length: { min: 10, max: 20 } }) },
        null,
      ]),
    }),
  ),
  total: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getListGrabsResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrabList, object>> = {},
): GrabsGrabList => ({
  grabs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    actions: {
      blocklistable: faker.datatype.boolean(),
      cancel: faker.datatype.boolean(),
      sendTo: faker.datatype.boolean(),
    },
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    downloadClient: faker.helpers.arrayElement([
      {
        id: faker.string.uuid(),
        kind: faker.helpers.arrayElement([
          "qbittorrent",
          "transmission",
          "deluge",
          "rtorrent",
        ] as const),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? {
          failure: {
            code: faker.helpers.arrayElement([
              "indexer_unavailable",
              "indexer_removed",
              "indexer_stopped",
              "home_offline",
              "torrent_unreachable",
              "magnet_invalid",
              "no_metadata",
              "no_peer_source",
              "torrent_invalid",
              "infohash_mismatch",
              "torrent_v2_only",
              "download_failed",
              "marked_failed",
              "removed",
              "no_video_file",
              "probe_failed",
              "fills_nothing",
              "name_taken",
              "name_too_long",
              "transfer_failed",
              "import_failed",
              "node_written_off",
              "root_changed",
              "canceled",
              "blocklisted",
              "release_fault",
            ] as const),
            message: faker.string.alpha({ length: { min: 10, max: 20 } }),
            needsYou: faker.datatype.boolean(),
            stage: faker.helpers.arrayElement([
              "fetching",
              "downloading",
              "probing",
              "transferring",
              "importing",
            ] as const),
          },
        }
      : {}),
    fetchNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    ...(faker.datatype.boolean()
      ? {
          fetchWait: {
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            until: faker.helpers.arrayElement([
              faker.date.past().toISOString().slice(0, 19) + "Z",
              null,
            ]),
          },
        }
      : {}),
    grabbedBy: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    history: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        stage: faker.helpers.arrayElement([
          "fetching",
          "downloading",
          "probing",
          "transferring",
          "importing",
          "imported",
          "failed",
        ] as const),
      }),
    ),
    id: faker.string.uuid(),
    indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    infoHash: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    itemId: faker.string.uuid(),
    manualSearchId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    ...(faker.datatype.boolean()
      ? {
          metadata: {
            endsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            peers: faker.number.int({ min: 0 }),
          },
        }
      : {}),
    nodeId: faker.string.uuid(),
    overrides: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { progress: { done: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) } }
      : {}),
    ...(faker.datatype.boolean()
      ? {
          quality: faker.helpers.arrayElement([
            "sdtv",
            "dvd",
            "webrip-480p",
            "hdtv-720p",
            "webrip-720p",
            "webdl-720p",
            "bluray-720p",
            "hdtv-1080p",
            "webrip-1080p",
            "webdl-1080p",
            "bluray-1080p",
            "remux-1080p",
            "hdtv-2160p",
            "webrip-2160p",
            "webdl-2160p",
            "bluray-2160p",
            "remux-2160p",
            "cam",
          ] as const),
        }
      : {}),
    releaseSize: faker.number.int({ min: 0 }),
    releaseTitle: faker.string.alpha({ length: { min: 10, max: 20 } }),
    releaseWords: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement(["search", "manual"] as const),
    stage: faker.helpers.arrayElement([
      "fetching",
      "downloading",
      "probing",
      "transferring",
      "importing",
      "imported",
      "failed",
    ] as const),
    targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    torrentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    trackerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    ...(faker.datatype.boolean()
      ? {
          waitsFor: {
            client: faker.helpers.arrayElement([
              {
                id: faker.string.uuid(),
                kind: faker.helpers.arrayElement([
                  "qbittorrent",
                  "transmission",
                  "deluge",
                  "rtorrent",
                ] as const),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            nodeId: faker.string.uuid(),
            off: faker.datatype.boolean(),
          },
        }
      : {}),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getGetGrabResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrab, object>> = {},
): GrabsGrab => ({
  actions: {
    blocklistable: faker.datatype.boolean(),
    cancel: faker.datatype.boolean(),
    sendTo: faker.datatype.boolean(),
  },
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  downloadClient: faker.helpers.arrayElement([
    {
      id: faker.string.uuid(),
      kind: faker.helpers.arrayElement([
        "qbittorrent",
        "transmission",
        "deluge",
        "rtorrent",
      ] as const),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? {
        failure: {
          code: faker.helpers.arrayElement([
            "indexer_unavailable",
            "indexer_removed",
            "indexer_stopped",
            "home_offline",
            "torrent_unreachable",
            "magnet_invalid",
            "no_metadata",
            "no_peer_source",
            "torrent_invalid",
            "infohash_mismatch",
            "torrent_v2_only",
            "download_failed",
            "marked_failed",
            "removed",
            "no_video_file",
            "probe_failed",
            "fills_nothing",
            "name_taken",
            "name_too_long",
            "transfer_failed",
            "import_failed",
            "node_written_off",
            "root_changed",
            "canceled",
            "blocklisted",
            "release_fault",
          ] as const),
          message: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needsYou: faker.datatype.boolean(),
          stage: faker.helpers.arrayElement([
            "fetching",
            "downloading",
            "probing",
            "transferring",
            "importing",
          ] as const),
        },
      }
    : {}),
  fetchNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...(faker.datatype.boolean()
    ? {
        fetchWait: {
          sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
          until: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
        },
      }
    : {}),
  grabbedBy: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  history: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      stage: faker.helpers.arrayElement([
        "fetching",
        "downloading",
        "probing",
        "transferring",
        "importing",
        "imported",
        "failed",
      ] as const),
    }),
  ),
  id: faker.string.uuid(),
  indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  infoHash: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  itemId: faker.string.uuid(),
  manualSearchId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...(faker.datatype.boolean()
    ? {
        metadata: {
          endsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          peers: faker.number.int({ min: 0 }),
        },
      }
    : {}),
  nodeId: faker.string.uuid(),
  overrides: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...(faker.datatype.boolean()
    ? { progress: { done: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) } }
    : {}),
  ...(faker.datatype.boolean()
    ? {
        quality: faker.helpers.arrayElement([
          "sdtv",
          "dvd",
          "webrip-480p",
          "hdtv-720p",
          "webrip-720p",
          "webdl-720p",
          "bluray-720p",
          "hdtv-1080p",
          "webrip-1080p",
          "webdl-1080p",
          "bluray-1080p",
          "remux-1080p",
          "hdtv-2160p",
          "webrip-2160p",
          "webdl-2160p",
          "bluray-2160p",
          "remux-2160p",
          "cam",
        ] as const),
      }
    : {}),
  releaseSize: faker.number.int({ min: 0 }),
  releaseTitle: faker.string.alpha({ length: { min: 10, max: 20 } }),
  releaseWords: faker.string.alpha({ length: { min: 10, max: 20 } }),
  source: faker.helpers.arrayElement(["search", "manual"] as const),
  stage: faker.helpers.arrayElement([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ] as const),
  targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  torrentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  trackerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...(faker.datatype.boolean()
    ? {
        waitsFor: {
          client: faker.helpers.arrayElement([
            {
              id: faker.string.uuid(),
              kind: faker.helpers.arrayElement([
                "qbittorrent",
                "transmission",
                "deluge",
                "rtorrent",
              ] as const),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          nodeId: faker.string.uuid(),
          off: faker.datatype.boolean(),
        },
      }
    : {}),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getListIndexerGrabsResponseMock = (
  overrideResponse: Partial<Extract<GrabsIndexerGrabList, object>> = {},
): GrabsIndexerGrabList => ({
  grabs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    actions: {
      blocklistable: faker.datatype.boolean(),
      cancel: faker.datatype.boolean(),
      sendTo: faker.datatype.boolean(),
    },
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    downloadClient: faker.helpers.arrayElement([
      {
        id: faker.string.uuid(),
        kind: faker.helpers.arrayElement([
          "qbittorrent",
          "transmission",
          "deluge",
          "rtorrent",
        ] as const),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? {
          failure: {
            code: faker.helpers.arrayElement([
              "indexer_unavailable",
              "indexer_removed",
              "indexer_stopped",
              "home_offline",
              "torrent_unreachable",
              "magnet_invalid",
              "no_metadata",
              "no_peer_source",
              "torrent_invalid",
              "infohash_mismatch",
              "torrent_v2_only",
              "download_failed",
              "marked_failed",
              "removed",
              "no_video_file",
              "probe_failed",
              "fills_nothing",
              "name_taken",
              "name_too_long",
              "transfer_failed",
              "import_failed",
              "node_written_off",
              "root_changed",
              "canceled",
              "blocklisted",
              "release_fault",
            ] as const),
            message: faker.string.alpha({ length: { min: 10, max: 20 } }),
            needsYou: faker.datatype.boolean(),
            stage: faker.helpers.arrayElement([
              "fetching",
              "downloading",
              "probing",
              "transferring",
              "importing",
            ] as const),
          },
        }
      : {}),
    fetchNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    ...(faker.datatype.boolean()
      ? {
          fetchWait: {
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            until: faker.helpers.arrayElement([
              faker.date.past().toISOString().slice(0, 19) + "Z",
              null,
            ]),
          },
        }
      : {}),
    grabbedBy: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    history: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        stage: faker.helpers.arrayElement([
          "fetching",
          "downloading",
          "probing",
          "transferring",
          "importing",
          "imported",
          "failed",
        ] as const),
      }),
    ),
    id: faker.string.uuid(),
    indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    infoHash: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    itemId: faker.string.uuid(),
    manualSearchId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    ...(faker.datatype.boolean()
      ? {
          metadata: {
            endsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            peers: faker.number.int({ min: 0 }),
          },
        }
      : {}),
    nodeId: faker.string.uuid(),
    overrides: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { progress: { done: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) } }
      : {}),
    ...(faker.datatype.boolean()
      ? {
          quality: faker.helpers.arrayElement([
            "sdtv",
            "dvd",
            "webrip-480p",
            "hdtv-720p",
            "webrip-720p",
            "webdl-720p",
            "bluray-720p",
            "hdtv-1080p",
            "webrip-1080p",
            "webdl-1080p",
            "bluray-1080p",
            "remux-1080p",
            "hdtv-2160p",
            "webrip-2160p",
            "webdl-2160p",
            "bluray-2160p",
            "remux-2160p",
            "cam",
          ] as const),
        }
      : {}),
    releaseSize: faker.number.int({ min: 0 }),
    releaseTitle: faker.string.alpha({ length: { min: 10, max: 20 } }),
    releaseWords: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement(["search", "manual"] as const),
    stage: faker.helpers.arrayElement([
      "fetching",
      "downloading",
      "probing",
      "transferring",
      "importing",
      "imported",
      "failed",
    ] as const),
    targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    torrentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    trackerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    ...(faker.datatype.boolean()
      ? {
          waitsFor: {
            client: faker.helpers.arrayElement([
              {
                id: faker.string.uuid(),
                kind: faker.helpers.arrayElement([
                  "qbittorrent",
                  "transmission",
                  "deluge",
                  "rtorrent",
                ] as const),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            nodeId: faker.string.uuid(),
            off: faker.datatype.boolean(),
          },
        }
      : {}),
    workId: faker.string.uuid(),
  })),
  nextOffset: faker.helpers.arrayElement([faker.number.int(), null]),
  ...overrideResponse,
});

export const getListSearchReleasesResponseMock = (
  overrideResponse: Partial<Extract<GrabsReleaseList, object>> = {},
): GrabsReleaseList => ({
  releases: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      copyOf: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      covers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      formatIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      indexerName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      lostAt: faker.helpers.arrayElement([
        {
          own: faker.string.alpha({ length: { min: 10, max: 20 } }),
          ref: faker.string.alpha({ length: { min: 10, max: 20 } }),
          step: faker.helpers.arrayElement([
            "want",
            "tier",
            "revision",
            "score",
            "priority",
            "freeleech",
            "seeders",
            "size",
            "title",
          ] as const),
          targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        },
        null,
      ]),
      outcome: faker.helpers.arrayElement([
        "grabbed",
        "not_taken",
        "outranked",
        "kept",
        "refused",
      ] as const),
      position: faker.number.int({ min: 0 }),
      rank: faker.helpers.arrayElement([faker.number.int({ min: 1 }), null]),
      reasons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean()
            ? {
                attribute: faker.helpers.arrayElement([
                  "hdr",
                  "video_codec",
                  "audio_codec",
                  "channels",
                  "language",
                ] as const),
              }
            : {}),
          code: faker.helpers.arrayElement([
            "wrong_work",
            "wrong_item",
            "covers_more",
            "wrong_title",
            "cut_not_wanted",
            "quality_not_in_target",
            "filter",
            "size_below",
            "size_above",
            "size_not_checked",
            "score_below_minimum",
            "not_an_upgrade",
            "repack_other_group",
            "blocklisted",
          ] as const),
          soft: faker.datatype.boolean(),
          ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
          ...(faker.datatype.boolean()
            ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
          words: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      scores: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({ score: faker.number.int(), targetId: faker.string.uuid() }),
      ),
      seeders: faker.number.int({ min: 0 }),
      size: faker.number.int({ min: 0 }),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getGetManualSearchResponseMock = (
  overrideResponse: Partial<Extract<GrabsManualSearch, object>> = {},
): GrabsManualSearch => ({
  folded: {
    groups: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fault: faker.datatype.boolean(),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    line: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  id: faker.string.uuid(),
  indexers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      automatic: faker.datatype.boolean(),
      done: faker.datatype.boolean(),
      failed: faker.datatype.boolean(),
      indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  line: faker.string.alpha({ length: { min: 10, max: 20 } }),
  pins: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    pin: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  plan: {
    headline: faker.string.alpha({ length: { min: 10, max: 20 } }),
    held: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        grabId: faker.string.uuid(),
        stage: faker.helpers.arrayElement([
          "fetching",
          "downloading",
          "probing",
          "transferring",
          "importing",
        ] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        ...(faker.datatype.boolean()
          ? { cause: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        choices: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            cells: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            edition: faker.helpers.arrayElement([
              {
                edition: faker.string.alpha({ length: { min: 1, max: 20 } }),
                words: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            lifts: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              ...(faker.datatype.boolean()
                ? {
                    attribute: faker.helpers.arrayElement([
                      "hdr",
                      "video_codec",
                      "audio_codec",
                      "channels",
                      "language",
                    ] as const),
                  }
                : {}),
              code: faker.helpers.arrayElement([
                "wrong_work",
                "wrong_item",
                "covers_more",
                "wrong_title",
                "cut_not_wanted",
                "quality_not_in_target",
                "filter",
                "size_below",
                "size_above",
                "size_not_checked",
                "score_below_minimum",
                "not_an_upgrade",
                "repack_other_group",
                "blocklisted",
              ] as const),
              soft: faker.datatype.boolean(),
              ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
              ...(faker.datatype.boolean()
                ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              words: faker.string.alpha({ length: { min: 10, max: 20 } }),
            })),
            ...(faker.datatype.boolean()
              ? { note: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            pin: faker.string.alpha({ length: { min: 10, max: 20 } }),
            release: {
              blocked: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              formatIds: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.uuid()),
              freeleech: faker.datatype.boolean(),
              id: faker.string.alpha({ length: { min: 10, max: 20 } }),
              indexers: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              lostAt: faker.helpers.arrayElement([
                {
                  own: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  ref: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  step: faker.helpers.arrayElement([
                    "want",
                    "tier",
                    "revision",
                    "score",
                    "priority",
                    "freeleech",
                    "seeders",
                    "size",
                    "title",
                  ] as const),
                  targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
                },
                null,
              ]),
              publishedAt: faker.helpers.arrayElement([
                faker.date.past().toISOString().slice(0, 19) + "Z",
                null,
              ]),
              seeders: faker.number.int({ min: 0 }),
              size: faker.number.int({ min: 0 }),
              title: faker.string.alpha({ length: { min: 10, max: 20 } }),
              ...(faker.datatype.boolean()
                ? { tracker: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              verdicts: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              words: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }),
        ),
        ...(faker.datatype.boolean()
          ? { failure: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? { missing: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { outcome: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        pick: faker.helpers.arrayElement([
          {
            cells: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            edition: faker.helpers.arrayElement([
              {
                edition: faker.string.alpha({ length: { min: 1, max: 20 } }),
                words: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            lifts: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => ({
              ...(faker.datatype.boolean()
                ? {
                    attribute: faker.helpers.arrayElement([
                      "hdr",
                      "video_codec",
                      "audio_codec",
                      "channels",
                      "language",
                    ] as const),
                  }
                : {}),
              code: faker.helpers.arrayElement([
                "wrong_work",
                "wrong_item",
                "covers_more",
                "wrong_title",
                "cut_not_wanted",
                "quality_not_in_target",
                "filter",
                "size_below",
                "size_above",
                "size_not_checked",
                "score_below_minimum",
                "not_an_upgrade",
                "repack_other_group",
                "blocklisted",
              ] as const),
              soft: faker.datatype.boolean(),
              ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
              ...(faker.datatype.boolean()
                ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              words: faker.string.alpha({ length: { min: 10, max: 20 } }),
            })),
            release: {
              blocked: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              formatIds: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.uuid()),
              freeleech: faker.datatype.boolean(),
              id: faker.string.alpha({ length: { min: 10, max: 20 } }),
              indexers: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              lostAt: faker.helpers.arrayElement([
                {
                  own: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  ref: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  step: faker.helpers.arrayElement([
                    "want",
                    "tier",
                    "revision",
                    "score",
                    "priority",
                    "freeleech",
                    "seeders",
                    "size",
                    "title",
                  ] as const),
                  targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
                },
                null,
              ]),
              publishedAt: faker.helpers.arrayElement([
                faker.date.past().toISOString().slice(0, 19) + "Z",
                null,
              ]),
              seeders: faker.number.int({ min: 0 }),
              size: faker.number.int({ min: 0 }),
              title: faker.string.alpha({ length: { min: 10, max: 20 } }),
              ...(faker.datatype.boolean()
                ? { tracker: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              verdicts: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              words: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            routed: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          null,
        ]),
        pinned: faker.datatype.boolean(),
      }),
    ),
  },
  query: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  queryable: faker.datatype.boolean(),
  releaseCount: faker.number.int({ min: 0 }),
  searching: faker.datatype.boolean(),
  startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  startedBy: faker.string.alpha({ length: { min: 10, max: 20 } }),
  title: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getStartManualSearchResponseMock = (
  overrideResponse: Partial<Extract<GrabsManualStarted, object>> = {},
): GrabsManualStarted =>
  faker.helpers.arrayElement([
    { manualSearchId: faker.string.uuid(), ...overrideResponse },
    { manualSearchId: faker.string.uuid(), ...overrideResponse },
  ]);

export const getGrabFromManualSearchResponseMock = (
  overrideResponse: Partial<Extract<GrabsPickResults, object>> = {},
): GrabsPickResults => ({
  results: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      grabId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      problem: faker.helpers.arrayElement([
        {
          code: faker.helpers.arrayElement([
            "pick_conflict",
            "cell_held",
            "pick_changed",
            "not_taken",
            "not_grabbable",
            "grab_busy",
            "download_client_field_refused",
            "node_not_downloading",
          ] as const),
          detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
          ...(faker.datatype.boolean() ? { grabId: faker.string.uuid() } : {}),
        },
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getListManualSearchReleasesResponseMock = (
  overrideResponse: Partial<Extract<GrabsOfferList, object>> = {},
): GrabsOfferList => ({
  releases: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      grab: {
        cancels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            grabId: faker.string.uuid(),
            stage: faker.helpers.arrayElement([
              "fetching",
              "downloading",
              "probing",
              "transferring",
              "importing",
            ] as const),
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        cells: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        edition: faker.helpers.arrayElement([
          {
            edition: faker.string.alpha({ length: { min: 1, max: 20 } }),
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          null,
        ]),
        grabbable: faker.datatype.boolean(),
        lifts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            ...(faker.datatype.boolean()
              ? {
                  attribute: faker.helpers.arrayElement([
                    "hdr",
                    "video_codec",
                    "audio_codec",
                    "channels",
                    "language",
                  ] as const),
                }
              : {}),
            code: faker.helpers.arrayElement([
              "wrong_work",
              "wrong_item",
              "covers_more",
              "wrong_title",
              "cut_not_wanted",
              "quality_not_in_target",
              "filter",
              "size_below",
              "size_above",
              "size_not_checked",
              "score_below_minimum",
              "not_an_upgrade",
              "repack_other_group",
              "blocklisted",
            ] as const),
            soft: faker.datatype.boolean(),
            ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
            ...(faker.datatype.boolean()
              ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        ...(faker.datatype.boolean()
          ? { outcome: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { refusal: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { waits: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
      },
      quality: faker.number.int({ min: 0 }),
      rank: faker.helpers.arrayElement([faker.number.int({ min: 1 }), null]),
      release: {
        blocked: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        formatIds: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => faker.string.uuid()),
        freeleech: faker.datatype.boolean(),
        id: faker.string.alpha({ length: { min: 10, max: 20 } }),
        indexers: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        lostAt: faker.helpers.arrayElement([
          {
            own: faker.string.alpha({ length: { min: 10, max: 20 } }),
            ref: faker.string.alpha({ length: { min: 10, max: 20 } }),
            step: faker.helpers.arrayElement([
              "want",
              "tier",
              "revision",
              "score",
              "priority",
              "freeleech",
              "seeders",
              "size",
              "title",
            ] as const),
            targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          },
          null,
        ]),
        publishedAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        seeders: faker.number.int({ min: 0 }),
        size: faker.number.int({ min: 0 }),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? { tracker: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        verdicts: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      score: faker.number.int(),
    }),
  ),
  ...overrideResponse,
});

export const getSearchWorkResponseMock = (
  overrideResponse: Partial<Extract<GrabsSearchStarted, object>> = {},
): GrabsSearchStarted => ({
  searchIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getListSearchesResponseMock = (
  overrideResponse: Partial<Extract<GrabsSearchList, object>> = {},
): GrabsSearchList => ({
  searches: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      candidates: faker.number.int({ min: 0 }),
      finishedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      grabId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      id: faker.string.uuid(),
      indexers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          outcome: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "found",
              "no_releases",
              "limited",
              "failing",
              "stopped",
              "waiting",
              "late",
              "limit_reached",
              "no_categories",
              "removed",
              "rss_only",
            ] as const),
            null,
          ]),
          releases: faker.number.int({ min: 0 }),
          sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      itemId: faker.string.uuid(),
      notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean()
            ? {
                attribute: faker.helpers.arrayElement([
                  "hdr",
                  "video_codec",
                  "audio_codec",
                  "channels",
                  "language",
                ] as const),
              }
            : {}),
          code: faker.helpers.arrayElement([
            "wrong_work",
            "wrong_item",
            "covers_more",
            "wrong_title",
            "cut_not_wanted",
            "quality_not_in_target",
            "filter",
            "size_below",
            "size_above",
            "size_not_checked",
            "score_below_minimum",
            "not_an_upgrade",
            "repack_other_group",
            "blocklisted",
          ] as const),
          soft: faker.datatype.boolean(),
          ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
          ...(faker.datatype.boolean()
            ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
        }),
      ),
      outcome: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["grabbed", "nothing_acceptable", "listed"] as const),
        null,
      ]),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  ...overrideResponse,
});

export const getListBlocklistMockHandler = (
  overrideResponse?:
    | GrabsBlocklist
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsBlocklist> | GrabsBlocklist),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/blocklist",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListBlocklistResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRemoveFromBlocklistMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/blocklist/remove",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListGrabsMockHandler = (
  overrideResponse?:
    | GrabsGrabList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsGrabList> | GrabsGrabList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/grabs",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListGrabsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetGrabMockHandler = (
  overrideResponse?:
    | GrabsGrab
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<GrabsGrab> | GrabsGrab),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/grabs/:grabId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetGrabResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelGrabMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/grabs/:grabId/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getSendGrabToMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/grabs/:grabId/send-to",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListIndexerGrabsMockHandler = (
  overrideResponse?:
    | GrabsIndexerGrabList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsIndexerGrabList> | GrabsIndexerGrabList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers/:id/grabs",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListIndexerGrabsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListSearchReleasesMockHandler = (
  overrideResponse?:
    | GrabsReleaseList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsReleaseList> | GrabsReleaseList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/searches/:searchId/releases",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListSearchReleasesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetManualSearchMockHandler = (
  overrideResponse?:
    | GrabsManualSearch
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsManualSearch> | GrabsManualSearch),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/manual-search",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetManualSearchResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartManualSearchMockHandler = (
  overrideResponse?:
    | GrabsManualStarted
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<GrabsManualStarted> | GrabsManualStarted),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/:workId/manual-search",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartManualSearchResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGrabFromManualSearchMockHandler = (
  overrideResponse?:
    | GrabsPickResults
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<GrabsPickResults> | GrabsPickResults),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/:workId/manual-search/grabs",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGrabFromManualSearchResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListManualSearchReleasesMockHandler = (
  overrideResponse?:
    | GrabsOfferList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsOfferList> | GrabsOfferList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/manual-search/releases",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListManualSearchReleasesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSearchWorkMockHandler = (
  overrideResponse?:
    | GrabsSearchStarted
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<GrabsSearchStarted> | GrabsSearchStarted),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/:workId/search",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSearchWorkResponseMock(),
        { status: 202 },
      );
    },
    options,
  );
};

export const getListSearchesMockHandler = (
  overrideResponse?:
    | GrabsSearchList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsSearchList> | GrabsSearchList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/searches",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListSearchesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getGrabsMock = () => [
  getListBlocklistMockHandler(),
  getRemoveFromBlocklistMockHandler(),
  getListGrabsMockHandler(),
  getGetGrabMockHandler(),
  getCancelGrabMockHandler(),
  getSendGrabToMockHandler(),
  getListIndexerGrabsMockHandler(),
  getListSearchReleasesMockHandler(),
  getGetManualSearchMockHandler(),
  getStartManualSearchMockHandler(),
  getGrabFromManualSearchMockHandler(),
  getListManualSearchReleasesMockHandler(),
  getSearchWorkMockHandler(),
  getListSearchesMockHandler(),
];
