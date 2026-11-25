import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  HomeSettings,
  HomeSummary,
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

export const getGetHomeResponseMock = (
  overrideResponse: Partial<Extract<HomeSummary, object>> = {},
): HomeSummary => ({
  jobs: {
    moving: faker.number.int({ min: 0 }),
    rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        grab: {
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
                  stage: faker.helpers.arrayElement([
                    "fetching",
                    "downloading",
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
          history: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            at: faker.date.past().toISOString().slice(0, 19) + "Z",
            stage: faker.helpers.arrayElement([
              "fetching",
              "downloading",
              "importing",
              "imported",
              "failed",
            ] as const),
          })),
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
          items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => faker.string.alpha({ length: { min: 10, max: 20 } }),
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
          overrides: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          ...(faker.datatype.boolean()
            ? {
                progress: {
                  done: faker.number.int({ min: 0 }),
                  total: faker.number.int({ min: 0 }),
                },
              }
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
          targetIds: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          targets: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            forced: faker.datatype.boolean(),
            itemId: faker.string.uuid(),
            settled: faker.helpers.arrayElement([
              faker.helpers.arrayElement(["placed", "none", "declined"] as const),
              null,
            ]),
            targetId: faker.string.uuid(),
          })),
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
        },
        torrent: faker.helpers.arrayElement([
          {
            addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            ...(faker.datatype.boolean()
              ? {
                  announce: {
                    host: faker.string.alpha({ length: { min: 10, max: 20 } }),
                    ...(faker.datatype.boolean()
                      ? { message: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                      : {}),
                    ...(faker.datatype.boolean()
                      ? { retryAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
                      : {}),
                    status: faker.helpers.arrayElement([
                      "working",
                      "updating",
                      "error",
                      "unregistered",
                      "not_contacted",
                    ] as const),
                  },
                }
              : {}),
            ...(faker.datatype.boolean() ? { attached: faker.datatype.boolean() } : {}),
            ...(faker.datatype.boolean() ? { clientId: faker.string.uuid() } : {}),
            ...(faker.datatype.boolean()
              ? {
                  clientKind: faker.helpers.arrayElement([
                    "qbittorrent",
                    "transmission",
                    "deluge",
                    "rtorrent",
                  ] as const),
                }
              : {}),
            ...(faker.datatype.boolean()
              ? { clientName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
            deleteFiles: faker.datatype.boolean(),
            done: faker.number.int({ min: 0 }),
            ...(faker.datatype.boolean()
              ? { edition: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            ...(faker.datatype.boolean()
              ? { error: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            ...(faker.datatype.boolean()
              ? {
                  fault: {
                    at: faker.date.past().toISOString().slice(0, 19) + "Z",
                    kind: faker.helpers.arrayElement([
                      "engine_error",
                      "unregistered",
                      "no_payload",
                    ] as const),
                    sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  },
                }
              : {}),
            held: faker.datatype.boolean(),
            ...(faker.datatype.boolean()
              ? {
                  hold: faker.helpers.arrayElement([
                    "wanted",
                    "window",
                    "disk_full",
                    "slot",
                    "check_slot",
                    "disk_guard",
                    "draining",
                    "controller",
                    "client_error",
                  ] as const),
                }
              : {}),
            id: faker.string.uuid(),
            ...(faker.datatype.boolean()
              ? { importAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
              : {}),
            ...(faker.datatype.boolean()
              ? { lastKnown: faker.date.past().toISOString().slice(0, 19) + "Z" }
              : {}),
            ...(faker.datatype.boolean()
              ? { lastKnownCause: faker.helpers.arrayElement(["node", "client", "off"] as const) }
              : {}),
            move: faker.helpers.arrayElement([
              {
                batchId: faker.string.uuid(),
                phase: faker.helpers.arrayElement([
                  "queued",
                  "transfer",
                  "place",
                  "switch",
                  "failed",
                ] as const),
                toNodeId: faker.string.uuid(),
                toNodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
                unitId: faker.string.uuid(),
                ...(faker.datatype.boolean()
                  ? {
                      waiting: faker.helpers.arrayElement([
                        "pair_busy",
                        "low_priority",
                        "node_offline",
                        "no_route",
                        "space",
                        "renames",
                        "downloading",
                        "importing",
                        "open_files",
                        "playing",
                        "media_servers",
                        "guard",
                        "linked_torrent",
                        "reconciling",
                      ] as const),
                    }
                  : {}),
                words: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            nodeId: faker.string.uuid(),
            ...(faker.datatype.boolean()
              ? { pending: faker.helpers.arrayElement(["pause", "resume"] as const) }
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
            ...(faker.datatype.boolean() ? { queuePosition: faker.number.int({ min: 0 }) } : {}),
            ratio: faker.number.float({ min: 0, fractionDigits: 2 }),
            ...(faker.datatype.boolean()
              ? {
                  retry: {
                    at: faker.date.past().toISOString().slice(0, 19) + "Z",
                    ...(faker.datatype.boolean() ? { found: faker.number.int({ min: 0 }) } : {}),
                  },
                }
              : {}),
            retryable: faker.datatype.boolean(),
            seeding: faker.helpers.arrayElement([
              {
                ...(faker.datatype.boolean()
                  ? {
                      action: faker.helpers.arrayElement([
                        "pause",
                        "remove",
                        "remove-delete",
                      ] as const),
                    }
                  : {}),
                atRisk: faker.helpers.arrayElement(["no", "yes", "unknown"] as const),
                clears: faker.helpers.arrayElement([
                  "none",
                  "time",
                  "ratio",
                  "either",
                  "both",
                ] as const),
                combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
                endless: faker.helpers.arrayElement([
                  "none",
                  "forever",
                  "resumed",
                  "no-tracker",
                  "no-rule",
                  "partial",
                ] as const),
                goalMet: faker.datatype.boolean(),
                ...(faker.datatype.boolean()
                  ? {
                      owes: {
                        bytes: faker.number.int({ min: 0 }),
                        seconds: faker.number.int({ min: 0 }),
                      },
                    }
                  : {}),
                ratio: {
                  ...(faker.datatype.boolean()
                    ? { goal: faker.number.float({ fractionDigits: 2 }) }
                    : {}),
                  ...(faker.datatype.boolean()
                    ? { rule: faker.number.float({ fractionDigits: 2 }) }
                    : {}),
                  value: faker.number.float({ min: 0, fractionDigits: 2 }),
                },
                state: faker.helpers.arrayElement([
                  "none",
                  "owing",
                  "met",
                  "missed",
                  "released",
                ] as const),
                time: {
                  ...(faker.datatype.boolean() ? { goal: faker.number.int({ min: 0 }) } : {}),
                  ...(faker.datatype.boolean() ? { margin: faker.number.int({ min: 0 }) } : {}),
                  ...(faker.datatype.boolean() ? { rule: faker.number.int({ min: 0 }) } : {}),
                  value: faker.number.int({ min: 0 }),
                },
              },
              null,
            ]),
            seedingSeconds: faker.number.int({ min: 0 }),
            severity: faker.helpers.arrayElement(["none", "warning", "failure"] as const),
            size: faker.number.int({ min: 0 }),
            state: faker.helpers.arrayElement([
              "removing",
              "failed",
              "tracker-error",
              "paused",
              "checking",
              "queued",
              "stalled",
              "downloading",
              "seeding",
            ] as const),
            ...(faker.datatype.boolean() ? { stoppedAtLimit: faker.datatype.boolean() } : {}),
            tabs: faker.helpers.arrayElements([
              "downloading",
              "seeding",
              "problems",
              "goal-met",
              "all",
            ] as const),
            tracker: faker.helpers.arrayElement([
              {
                id: faker.string.uuid(),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            ...(faker.datatype.boolean()
              ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
              : {}),
            uploaded: faker.number.int({ min: 0 }),
            ...(faker.datatype.boolean() ? { workId: faker.string.uuid() } : {}),
          },
          null,
        ]),
        work: {
          backdropUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
          failed: faker.datatype.boolean(),
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        },
      }),
    ),
  },
  needsYou: {
    failed: faker.number.int({ min: 0, max: 100 }),
    moves: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        batchId: faker.string.uuid(),
        kind: faker.helpers.arrayElement(["failed", "waiting"] as const),
        ...(faker.datatype.boolean()
          ? { since: faker.date.past().toISOString().slice(0, 19) + "Z" }
          : {}),
        switchNow: faker.datatype.boolean(),
        unitId: faker.string.uuid(),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
        work: {
          backdropUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
          failed: faker.datatype.boolean(),
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        },
      }),
    ),
    rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        grab: {
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
                  stage: faker.helpers.arrayElement([
                    "fetching",
                    "downloading",
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
          history: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            at: faker.date.past().toISOString().slice(0, 19) + "Z",
            stage: faker.helpers.arrayElement([
              "fetching",
              "downloading",
              "importing",
              "imported",
              "failed",
            ] as const),
          })),
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
          items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => faker.string.alpha({ length: { min: 10, max: 20 } }),
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
          overrides: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          ...(faker.datatype.boolean()
            ? {
                progress: {
                  done: faker.number.int({ min: 0 }),
                  total: faker.number.int({ min: 0 }),
                },
              }
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
          targetIds: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          targets: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            forced: faker.datatype.boolean(),
            itemId: faker.string.uuid(),
            settled: faker.helpers.arrayElement([
              faker.helpers.arrayElement(["placed", "none", "declined"] as const),
              null,
            ]),
            targetId: faker.string.uuid(),
          })),
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
        },
        wait: faker.helpers.arrayElement([
          {
            clientError: faker.helpers.arrayElement([
              {
                client: faker.string.alpha({ length: { min: 10, max: 20 } }),
                failsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
                message: faker.string.alpha({ length: { min: 10, max: 20 } }),
                node: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            code: faker.helpers.arrayElement([
              "node_away",
              "client_away",
              "client_off",
              "no_space",
              "root_not_ready",
              "probe_timeout",
              "probe_error",
              "client_error",
            ] as const),
            grabId: faker.string.uuid(),
            importWait: faker.helpers.arrayElement([
              {
                rootId: faker.string.uuid(),
                space: faker.helpers.arrayElement([
                  { free: faker.number.int({ min: 0 }), needs: faker.number.int({ min: 1 }) },
                  null,
                ]),
              },
              null,
            ]),
            kind: faker.helpers.arrayElement(["waiting"] as const),
          },
          null,
        ]),
        work: {
          backdropUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
          failed: faker.datatype.boolean(),
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        },
      }),
    ),
    waiting: faker.number.int({ min: 0 }),
  },
  recent: {
    rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        importId: faker.string.uuid(),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
        work: {
          backdropUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
          failed: faker.datatype.boolean(),
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        },
      }),
    ),
    today: { imported: faker.number.int({ min: 0 }), upgraded: faker.number.int({ min: 0 }) },
  },
  ...overrideResponse,
});

export const getGetHomeSettingsResponseSettingDurationFieldMock = (
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

export const getGetHomeSettingsResponseSettingCountFieldMock = (
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

export const getGetHomeSettingsResponseSettingChoiceFieldMock = (
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

export const getGetHomeSettingsResponseSettingWindowFieldMock = (
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

export const getGetHomeSettingsResponseSettingListenFieldMock = (
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

export const getGetHomeSettingsResponseSettingSwitchFieldMock = (
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

export const getGetHomeSettingsResponseSettingRateFieldMock = (
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

export const getGetHomeSettingsResponseSettingAddressFieldMock = (
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

export const getGetHomeSettingsResponseSettingFolderFieldMock = (
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

export const getGetHomeSettingsResponseSettingSpaceFieldMock = (
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

export const getGetHomeSettingsResponseSettingTemplateFieldMock = (
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

export const getGetHomeSettingsResponseSettingListFieldMock = (
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

export const getGetHomeSettingsResponseSettingSizeFieldMock = (
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

export const getGetHomeSettingsResponseSettingIPFieldMock = (
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

export const getGetHomeSettingsResponseMock = (
  overrideResponse: Partial<Extract<HomeSettings, object>> = {},
): HomeSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetHomeSettingsResponseSettingDurationFieldMock() },
      { ...getGetHomeSettingsResponseSettingCountFieldMock() },
      { ...getGetHomeSettingsResponseSettingChoiceFieldMock() },
      { ...getGetHomeSettingsResponseSettingWindowFieldMock() },
      { ...getGetHomeSettingsResponseSettingListenFieldMock() },
      { ...getGetHomeSettingsResponseSettingSwitchFieldMock() },
      { ...getGetHomeSettingsResponseSettingRateFieldMock() },
      { ...getGetHomeSettingsResponseSettingAddressFieldMock() },
      { ...getGetHomeSettingsResponseSettingFolderFieldMock() },
      { ...getGetHomeSettingsResponseSettingSpaceFieldMock() },
      { ...getGetHomeSettingsResponseSettingTemplateFieldMock() },
      { ...getGetHomeSettingsResponseSettingListFieldMock() },
      { ...getGetHomeSettingsResponseSettingSizeFieldMock() },
      { ...getGetHomeSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateHomeSettingsResponseSettingDurationFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingCountFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingChoiceFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingWindowFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingListenFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingSwitchFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingRateFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingAddressFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingFolderFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingSpaceFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingTemplateFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingListFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingSizeFieldMock = (
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

export const getUpdateHomeSettingsResponseSettingIPFieldMock = (
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

export const getUpdateHomeSettingsResponseMock = (
  overrideResponse: Partial<Extract<HomeSettings, object>> = {},
): HomeSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateHomeSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingCountFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingListenFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingRateFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingSpaceFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingTemplateFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingListFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingSizeFieldMock() },
      { ...getUpdateHomeSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getGetHomeMockHandler = (
  overrideResponse?:
    | HomeSummary
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<HomeSummary> | HomeSummary),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/home",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetHomeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetHomeSettingsMockHandler = (
  overrideResponse?:
    | HomeSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HomeSettings> | HomeSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/home-settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetHomeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateHomeSettingsMockHandler = (
  overrideResponse?:
    | HomeSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<HomeSettings> | HomeSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/home-settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateHomeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getHomeMock = () => [
  getGetHomeMockHandler(),
  getGetHomeSettingsMockHandler(),
  getUpdateHomeSettingsMockHandler(),
];
