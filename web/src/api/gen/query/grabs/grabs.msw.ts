import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  GrabsBlocklist,
  GrabsGrab,
  GrabsGrabList,
  GrabsGrabbing,
  GrabsIndexerGrabList,
  GrabsManualSearch,
  GrabsManualStarted,
  GrabsOfferList,
  GrabsPickResults,
  GrabsQueuedCanceled,
  GrabsReleaseList,
  GrabsSearchList,
  GrabsSearchQueued,
  GrabsSearchSettings,
  GrabsSeveralQueued,
  GrabsWantedPage,
  GrabsWorkSearching,
  SettingAddressField,
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingFolderField,
  SettingIPField,
  SettingListField,
  SettingListenField,
  SettingRateField,
  SettingSizeField,
  SettingSpaceField,
  SettingSwitchField,
  SettingTemplateField,
  SettingWindowField,
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
        "no_video",
        "unreadable",
        "misdescribed",
      ] as const),
      facts: {
        ...(faker.datatype.boolean()
          ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
        ...(faker.datatype.boolean()
          ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
      },
      indexer: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      infoHash: faker.helpers.fromRegExp("^[0-9a-f]{40}$"),
      size: faker.number.int({ min: 0 }),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
    chosenBecause: faker.helpers.arrayElement([
      {
        fills: faker.number.int({ min: 1 }),
        kind: faker.helpers.arrayElement(["fewer_releases"] as const),
      },
      null,
    ]),
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
            cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
              "client_removed",
              "client_deleted",
              "goal_met",
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
              "client_error",
              "declined",
              "unreadable",
              "restored",
            ] as const),
            current: faker.datatype.boolean(),
            message: faker.string.alpha({ length: { min: 10, max: 20 } }),
            needsYou: faker.datatype.boolean(),
            recovery: faker.helpers.arrayElement([
              faker.string.alpha({ length: { min: 10, max: 20 } }),
              null,
            ]),
            stage: faker.helpers.arrayElement(["fetching", "downloading", "importing"] as const),
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
    foundBy: faker.helpers.arrayElement([
      "search",
      "backlog",
      "rss",
      "failed_grab",
      "manual",
    ] as const),
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
          "importing",
          "imported",
          "failed",
        ] as const),
      }),
    ),
    id: faker.string.uuid(),
    import: faker.helpers.arrayElement([
      {
        id: faker.string.uuid(),
        placed: faker.number.int({ min: 0 }),
        planned: faker.number.int({ min: 0 }),
        sending: faker.number.int({ min: 0 }),
        waiting: faker.number.int({ min: 0 }),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    inFlight: faker.datatype.boolean(),
    indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    infoHash: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
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
    releaseFacts: {
      ...(faker.datatype.boolean()
        ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
      ...(faker.datatype.boolean()
        ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    },
    releaseSize: faker.number.int({ min: 0 }),
    releaseTitle: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rssIndexerName: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    searchId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    source: faker.helpers.arrayElement(["search", "manual", "rss"] as const),
    stage: faker.helpers.arrayElement([
      "fetching",
      "downloading",
      "importing",
      "imported",
      "failed",
    ] as const),
    targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        forced: faker.datatype.boolean(),
        itemId: faker.string.uuid(),
        settled: faker.helpers.arrayElement([
          faker.helpers.arrayElement(["placed", "none", "declined"] as const),
          null,
        ]),
        targetId: faker.string.uuid(),
      }),
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
  chosenBecause: faker.helpers.arrayElement([
    {
      fills: faker.number.int({ min: 1 }),
      kind: faker.helpers.arrayElement(["fewer_releases"] as const),
    },
    null,
  ]),
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
          cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
            "client_removed",
            "client_deleted",
            "goal_met",
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
            "client_error",
            "declined",
            "unreadable",
            "restored",
          ] as const),
          current: faker.datatype.boolean(),
          message: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needsYou: faker.datatype.boolean(),
          recovery: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          stage: faker.helpers.arrayElement(["fetching", "downloading", "importing"] as const),
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
  foundBy: faker.helpers.arrayElement([
    "search",
    "backlog",
    "rss",
    "failed_grab",
    "manual",
  ] as const),
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
        "importing",
        "imported",
        "failed",
      ] as const),
    }),
  ),
  id: faker.string.uuid(),
  import: faker.helpers.arrayElement([
    {
      id: faker.string.uuid(),
      placed: faker.number.int({ min: 0 }),
      planned: faker.number.int({ min: 0 }),
      sending: faker.number.int({ min: 0 }),
      waiting: faker.number.int({ min: 0 }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  inFlight: faker.datatype.boolean(),
  indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  infoHash: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
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
  releaseFacts: {
    ...(faker.datatype.boolean()
      ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
    ...(faker.datatype.boolean()
      ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  releaseSize: faker.number.int({ min: 0 }),
  releaseTitle: faker.string.alpha({ length: { min: 10, max: 20 } }),
  rssIndexerName: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  searchId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  source: faker.helpers.arrayElement(["search", "manual", "rss"] as const),
  stage: faker.helpers.arrayElement([
    "fetching",
    "downloading",
    "importing",
    "imported",
    "failed",
  ] as const),
  targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      forced: faker.datatype.boolean(),
      itemId: faker.string.uuid(),
      settled: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["placed", "none", "declined"] as const),
        null,
      ]),
      targetId: faker.string.uuid(),
    }),
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
    chosenBecause: faker.helpers.arrayElement([
      {
        fills: faker.number.int({ min: 1 }),
        kind: faker.helpers.arrayElement(["fewer_releases"] as const),
      },
      null,
    ]),
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
            cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
              "client_removed",
              "client_deleted",
              "goal_met",
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
              "client_error",
              "declined",
              "unreadable",
              "restored",
            ] as const),
            current: faker.datatype.boolean(),
            message: faker.string.alpha({ length: { min: 10, max: 20 } }),
            needsYou: faker.datatype.boolean(),
            recovery: faker.helpers.arrayElement([
              faker.string.alpha({ length: { min: 10, max: 20 } }),
              null,
            ]),
            stage: faker.helpers.arrayElement(["fetching", "downloading", "importing"] as const),
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
    foundBy: faker.helpers.arrayElement([
      "search",
      "backlog",
      "rss",
      "failed_grab",
      "manual",
    ] as const),
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
          "importing",
          "imported",
          "failed",
        ] as const),
      }),
    ),
    id: faker.string.uuid(),
    import: faker.helpers.arrayElement([
      {
        id: faker.string.uuid(),
        placed: faker.number.int({ min: 0 }),
        planned: faker.number.int({ min: 0 }),
        sending: faker.number.int({ min: 0 }),
        waiting: faker.number.int({ min: 0 }),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    inFlight: faker.datatype.boolean(),
    indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    infoHash: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
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
    releaseFacts: {
      ...(faker.datatype.boolean()
        ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
      ...(faker.datatype.boolean()
        ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    },
    releaseSize: faker.number.int({ min: 0 }),
    releaseTitle: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rssIndexerName: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    searchId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    source: faker.helpers.arrayElement(["search", "manual", "rss"] as const),
    stage: faker.helpers.arrayElement([
      "fetching",
      "downloading",
      "importing",
      "imported",
      "failed",
    ] as const),
    targetIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        forced: faker.datatype.boolean(),
        itemId: faker.string.uuid(),
        settled: faker.helpers.arrayElement([
          faker.helpers.arrayElement(["placed", "none", "declined"] as const),
          null,
        ]),
        targetId: faker.string.uuid(),
      }),
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

export const getCancelQueuedSearchesResponseMock = (
  overrideResponse: Partial<Extract<GrabsQueuedCanceled, object>> = {},
): GrabsQueuedCanceled => ({ canceled: faker.number.int({ min: 0 }), ...overrideResponse });

export const getGetSearchSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { description: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingListenFieldMock = (
  overrideResponse: Partial<SettingListenField> = {},
): SettingListenField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      host: faker.helpers.arrayElement(["all", "local", "other"] as const),
      port: faker.number.int(),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["listen"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      host: faker.helpers.arrayElement(["all", "local", "other"] as const),
      port: faker.number.int(),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      host: faker.helpers.arrayElement(["all", "local", "other"] as const),
      port: faker.number.int(),
    },
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads", "backups"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingTemplateFieldMock = (
  overrideResponse: Partial<SettingTemplateField> = {},
): SettingTemplateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    field: faker.string.alpha({ length: { min: 10, max: 20 } }),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["template"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    plain: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingListFieldMock = (
  overrideResponse: Partial<SettingListField> = {},
): SettingListField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    item: faker.helpers.arrayElement(["extension", "host-port"] as const),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["list"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    min: faker.number.int({ min: 0 }),
    pendingRestart: faker.datatype.boolean(),
    running: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingSizeFieldMock = (
  overrideResponse: Partial<SettingSizeField> = {},
): SettingSizeField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["size"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseSettingIPFieldMock = (
  overrideResponse: Partial<SettingIPField> = {},
): SettingIPField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    family: faker.helpers.arrayElement(["any", "ipv6"] as const),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["ip"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  },
  ...overrideResponse,
});

export const getGetSearchSettingsResponseMock = (
  overrideResponse: Partial<Extract<GrabsSearchSettings, object>> = {},
): GrabsSearchSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetSearchSettingsResponseSettingDurationFieldMock() },
      { ...getGetSearchSettingsResponseSettingCountFieldMock() },
      { ...getGetSearchSettingsResponseSettingChoiceFieldMock() },
      { ...getGetSearchSettingsResponseSettingWindowFieldMock() },
      { ...getGetSearchSettingsResponseSettingListenFieldMock() },
      { ...getGetSearchSettingsResponseSettingSwitchFieldMock() },
      { ...getGetSearchSettingsResponseSettingRateFieldMock() },
      { ...getGetSearchSettingsResponseSettingAddressFieldMock() },
      { ...getGetSearchSettingsResponseSettingFolderFieldMock() },
      { ...getGetSearchSettingsResponseSettingSpaceFieldMock() },
      { ...getGetSearchSettingsResponseSettingTemplateFieldMock() },
      { ...getGetSearchSettingsResponseSettingListFieldMock() },
      { ...getGetSearchSettingsResponseSettingSizeFieldMock() },
      { ...getGetSearchSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { description: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["count"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingListenFieldMock = (
  overrideResponse: Partial<SettingListenField> = {},
): SettingListenField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      host: faker.helpers.arrayElement(["all", "local", "other"] as const),
      port: faker.number.int(),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["listen"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      host: faker.helpers.arrayElement(["all", "local", "other"] as const),
      port: faker.number.int(),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      host: faker.helpers.arrayElement(["all", "local", "other"] as const),
      port: faker.number.int(),
    },
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads", "backups"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingTemplateFieldMock = (
  overrideResponse: Partial<SettingTemplateField> = {},
): SettingTemplateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    field: faker.string.alpha({ length: { min: 10, max: 20 } }),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["template"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    plain: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingListFieldMock = (
  overrideResponse: Partial<SettingListField> = {},
): SettingListField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    item: faker.helpers.arrayElement(["extension", "host-port"] as const),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["list"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    min: faker.number.int({ min: 0 }),
    pendingRestart: faker.datatype.boolean(),
    running: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingSizeFieldMock = (
  overrideResponse: Partial<SettingSizeField> = {},
): SettingSizeField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["size"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseSettingIPFieldMock = (
  overrideResponse: Partial<SettingIPField> = {},
): SettingIPField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    family: faker.helpers.arrayElement(["any", "ipv6"] as const),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["ip"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  },
  ...overrideResponse,
});

export const getUpdateSearchSettingsResponseMock = (
  overrideResponse: Partial<Extract<GrabsSearchSettings, object>> = {},
): GrabsSearchSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateSearchSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingCountFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingListenFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingRateFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingSpaceFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingTemplateFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingListFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingSizeFieldMock() },
      { ...getUpdateSearchSettingsResponseSettingIPFieldMock() },
    ]),
  ),
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
      facts: {
        ...(faker.datatype.boolean()
          ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
        ...(faker.datatype.boolean()
          ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
      },
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
            "fewer_releases",
            "tier",
            "revision",
            "score",
            "chosen_by_another",
            "fewer_unwanted",
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
        "waiting",
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
            "not_aired",
            "tried",
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
    }),
  ),
  ...overrideResponse,
});

export const getListWantedResponseMock = (
  overrideResponse: Partial<Extract<GrabsWantedPage, object>> = {},
): GrabsWantedPage => ({
  leftGone: faker.number.int({ min: 0 }),
  next: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    lastResult: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "grabbed",
        "waiting",
        "nothing_found",
        "nothing_acceptable",
        "no_indexer",
        "no_root",
      ] as const),
      null,
    ]),
    lastSearchedAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    nextSearchAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    others: {
      [faker.string.alphanumeric(5)]: faker.number.int(),
    },
    ranges: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        first: faker.string.alpha({ length: { min: 10, max: 20 } }),
        last: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    reason: faker.helpers.arrayElement(["failed", "missing", "upgrade", "gone"] as const),
    state: faker.helpers.arrayElement(["idle", "queued", "searching"] as const),
    work: {
      contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      id: faker.string.uuid(),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      posterUrl: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
    },
  })),
  status: {
    limits: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        indexerId: faker.string.uuid(),
        indexerName: faker.string.alpha({ length: { min: 10, max: 20 } }),
        starved: faker.datatype.boolean(),
        until: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    nextBacklogAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    queued: faker.number.int({ min: 0 }),
    searching: faker.number.int({ min: 0 }),
  },
  total: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getSearchSeveralWorksResponseMock = (
  overrideResponse: Partial<Extract<GrabsSeveralQueued, object>> = {},
): GrabsSeveralQueued => ({
  leftGone: faker.number.int({ min: 0 }),
  merged: faker.number.int({ min: 0 }),
  queued: faker.number.int({ min: 0 }),
  works: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    outcome: faker.helpers.arrayElement(["queued", "merged", "skipped"] as const),
    reason: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "nothing_wanted",
        "gone_only",
        "no_root",
        "not_searchable",
      ] as const),
      null,
    ]),
    workId: faker.string.uuid(),
  })),
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
        stage: faker.helpers.arrayElement(["fetching", "downloading", "importing"] as const),
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
                "not_aired",
                "tried",
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
              facts: {
                ...(faker.datatype.boolean()
                  ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
                ...(faker.datatype.boolean()
                  ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
              },
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
                    "fewer_releases",
                    "tier",
                    "revision",
                    "score",
                    "chosen_by_another",
                    "fewer_unwanted",
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
                "not_aired",
                "tried",
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
              facts: {
                ...(faker.datatype.boolean()
                  ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
                ...(faker.datatype.boolean()
                  ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
                ...(faker.datatype.boolean()
                  ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                  : {}),
              },
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
                    "fewer_releases",
                    "tier",
                    "revision",
                    "score",
                    "chosen_by_another",
                    "fewer_unwanted",
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
            },
            ...(faker.datatype.boolean()
              ? { routed: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
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
  scope: {
    ...(faker.datatype.boolean()
      ? {
          cell: {
            edition: faker.string.alpha({ length: { min: 1, max: 20 } }),
            itemId: faker.string.uuid(),
            targetId: faker.string.uuid(),
          },
        }
      : {}),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.uuid(),
    ),
    levelIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.uuid(),
    ),
  },
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
            stage: faker.helpers.arrayElement(["fetching", "downloading", "importing"] as const),
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
              "not_aired",
              "tried",
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
        facts: {
          ...(faker.datatype.boolean()
            ? { audio: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
          ...(faker.datatype.boolean()
            ? { group: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
          ...(faker.datatype.boolean()
            ? { hdr: faker.string.alpha({ length: { min: 10, max: 20 } }) }
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
          ...(faker.datatype.boolean()
            ? { revision: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
        },
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
              "fewer_releases",
              "tier",
              "revision",
              "score",
              "chosen_by_another",
              "fewer_unwanted",
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
      },
      score: faker.number.int(),
    }),
  ),
  ...overrideResponse,
});

export const getSearchWorkResponseMock = (
  overrideResponse: Partial<Extract<GrabsSearchQueued, object>> = {},
): GrabsSearchQueued => ({
  groups: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      state: faker.helpers.arrayElement(["queued", "merged"] as const),
    }),
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
      grabIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      grabs: faker.number.int({ min: 0 }),
      groupKey: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      groupLabel: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
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
      itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      lane: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["now", "bulk", "backlog"] as const),
        null,
      ]),
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
            "not_aired",
            "tried",
          ] as const),
          soft: faker.datatype.boolean(),
          ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
          ...(faker.datatype.boolean()
            ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
        }),
      ),
      outcome: faker.helpers.arrayElement([
        faker.helpers.arrayElement([
          "grabbed",
          "waiting",
          "nothing_found",
          "nothing_acceptable",
          "no_indexer",
          "no_root",
          "listed",
        ] as const),
        null,
      ]),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      trigger: faker.helpers.arrayElement([
        faker.helpers.arrayElement([
          "search",
          "search_all",
          "add",
          "monitor",
          "profile",
          "editions",
          "retarget",
          "release_date",
          "refresh",
          "backlog",
          "failed_grab",
          "rss",
        ] as const),
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getGetWorkSearchingResponseMock = (
  overrideResponse: Partial<Extract<GrabsWorkSearching, object>> = {},
): GrabsWorkSearching => ({
  groups: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      again: faker.datatype.boolean(),
      behind: faker.helpers.arrayElement([
        {
          ahead: faker.number.int({ min: 0 }),
          aheadFrom: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "search",
              "search_all",
              "add",
              "monitor",
              "profile",
              "editions",
              "retarget",
              "release_date",
              "refresh",
              "backlog",
              "failed_grab",
            ] as const),
            null,
          ]),
          aheadLane: faker.helpers.arrayElement([
            faker.helpers.arrayElement(["now", "bulk", "backlog"] as const),
            null,
          ]),
          indexerId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          indexerName: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          starved: faker.datatype.boolean(),
          until: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
        },
        null,
      ]),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      lane: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["now", "bulk", "backlog"] as const),
        null,
      ]),
      lastFound: faker.number.int({ min: 0 }),
      lastGrabs: faker.number.int({ min: 0 }),
      lastResult: faker.helpers.arrayElement([
        faker.helpers.arrayElement([
          "grabbed",
          "waiting",
          "nothing_found",
          "nothing_acceptable",
          "no_indexer",
          "no_root",
        ] as const),
        null,
      ]),
      lastSearchedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      nextSearchAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      state: faker.helpers.arrayElement(["idle", "queued", "searching"] as const),
      trigger: faker.helpers.arrayElement([
        faker.helpers.arrayElement([
          "search",
          "search_all",
          "add",
          "monitor",
          "profile",
          "editions",
          "retarget",
          "release_date",
          "refresh",
          "backlog",
          "failed_grab",
        ] as const),
        null,
      ]),
    }),
  ),
  lead: {
    groupKey: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    of: faker.number.int({ min: 0 }),
    wait: faker.helpers.arrayElement([
      {
        candidates: faker.number.int({ min: 1 }),
        cell: {
          edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
          itemId: faker.string.uuid(),
          targetId: faker.string.uuid(),
        },
        until: faker.date.past().toISOString().slice(0, 19) + "Z",
      },
      null,
    ]),
  },
  stops: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    cell: {
      edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
      itemId: faker.string.uuid(),
      targetId: faker.string.uuid(),
    },
    failures: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
          "client_removed",
          "client_deleted",
          "goal_met",
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
          "client_error",
          "declined",
          "unreadable",
          "restored",
        ] as const),
        grabId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      }),
    ),
    nextSearchAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
  })),
  waits: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    candidates: faker.number.int({ min: 1 }),
    cell: {
      edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
      itemId: faker.string.uuid(),
      targetId: faker.string.uuid(),
    },
    until: faker.date.past().toISOString().slice(0, 19) + "Z",
  })),
  ...overrideResponse,
});

export const getGrabWaitingResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrabbing, object>> = {},
): GrabsGrabbing => ({ grabbing: faker.number.int({ min: 0 }), ...overrideResponse });

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

export const getCancelQueuedSearchesMockHandler = (
  overrideResponse?:
    | GrabsQueuedCanceled
    | ((
        info: Parameters<Parameters<typeof http.delete>[1]>[0],
      ) => Promise<GrabsQueuedCanceled> | GrabsQueuedCanceled),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/search-queue",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelQueuedSearchesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetSearchSettingsMockHandler = (
  overrideResponse?:
    | GrabsSearchSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsSearchSettings> | GrabsSearchSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/search-settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSearchSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateSearchSettingsMockHandler = (
  overrideResponse?:
    | GrabsSearchSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<GrabsSearchSettings> | GrabsSearchSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/search-settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateSearchSettingsResponseMock(),
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

export const getListWantedMockHandler = (
  overrideResponse?:
    | GrabsWantedPage
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsWantedPage> | GrabsWantedPage),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/wanted",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListWantedResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSearchSeveralWorksMockHandler = (
  overrideResponse?:
    | GrabsSeveralQueued
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<GrabsSeveralQueued> | GrabsSeveralQueued),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/search",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSearchSeveralWorksResponseMock(),
        { status: 202 },
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
    | GrabsSearchQueued
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<GrabsSearchQueued> | GrabsSearchQueued),
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

export const getGetWorkSearchingMockHandler = (
  overrideResponse?:
    | GrabsWorkSearching
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<GrabsWorkSearching> | GrabsWorkSearching),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/searching",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkSearchingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGrabWaitingMockHandler = (
  overrideResponse?:
    | GrabsGrabbing
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<GrabsGrabbing> | GrabsGrabbing),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/:workId/waits/grab",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGrabWaitingResponseMock(),
        { status: 202 },
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
  getCancelQueuedSearchesMockHandler(),
  getGetSearchSettingsMockHandler(),
  getUpdateSearchSettingsMockHandler(),
  getListSearchReleasesMockHandler(),
  getListWantedMockHandler(),
  getSearchSeveralWorksMockHandler(),
  getGetManualSearchMockHandler(),
  getStartManualSearchMockHandler(),
  getGrabFromManualSearchMockHandler(),
  getListManualSearchReleasesMockHandler(),
  getSearchWorkMockHandler(),
  getListSearchesMockHandler(),
  getGetWorkSearchingMockHandler(),
  getGrabWaitingMockHandler(),
];
