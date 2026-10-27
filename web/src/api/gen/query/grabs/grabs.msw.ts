import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { GrabsGrab, GrabsGrabList, GrabsSearchList, GrabsSearchStarted } from "../../model";

export const getListGrabsResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrabList, object>> = {},
): GrabsGrabList => ({
  grabs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
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
              "no_download_client",
              "download_failed",
              "removed",
              "no_video_file",
              "probe_failed",
              "fills_nothing",
              "transfer_failed",
              "import_failed",
              "node_written_off",
              "root_changed",
              "needs_transfer",
            ] as const),
            message: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
    ...(faker.datatype.boolean()
      ? {
          metadata: {
            endsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            peers: faker.number.int({ min: 0 }),
          },
        }
      : {}),
    nodeId: faker.string.uuid(),
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
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getGetGrabResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrab, object>> = {},
): GrabsGrab => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
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
            "no_download_client",
            "download_failed",
            "removed",
            "no_video_file",
            "probe_failed",
            "fills_nothing",
            "transfer_failed",
            "import_failed",
            "node_written_off",
            "root_changed",
            "needs_transfer",
          ] as const),
          message: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
  ...(faker.datatype.boolean()
    ? {
        metadata: {
          endsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          peers: faker.number.int({ min: 0 }),
        },
      }
    : {}),
  nodeId: faker.string.uuid(),
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
  workId: faker.string.uuid(),
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
            "cut_not_wanted",
            "quality_not_in_target",
            "filter",
            "size_below",
            "size_above",
            "size_not_checked",
            "not_an_upgrade",
          ] as const),
          ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
          ...(faker.datatype.boolean()
            ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
        }),
      ),
      outcome: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["grabbed", "nothing_acceptable"] as const),
        null,
      ]),
      refused: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          reasons: Array.from(
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
              "cut_not_wanted",
              "quality_not_in_target",
              "filter",
              "size_below",
              "size_above",
              "size_not_checked",
              "not_an_upgrade",
            ] as const),
            ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
            ...(faker.datatype.boolean()
              ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
          })),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  ...overrideResponse,
});

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
  getListGrabsMockHandler(),
  getGetGrabMockHandler(),
  getSearchWorkMockHandler(),
  getListSearchesMockHandler(),
];
