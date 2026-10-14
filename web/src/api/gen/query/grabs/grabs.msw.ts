import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { GrabsGrab, GrabsGrabList, GrabsSearchList, GrabsSearchStarted } from "../../model";

export const getListGrabsResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrabList, object>> = {},
): GrabsGrabList => ({
  grabs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    edition: faker.helpers.arrayElement(["standard"] as const),
    ...(faker.datatype.boolean()
      ? {
          failure: {
            code: faker.helpers.arrayElement([
              "indexer_unavailable",
              "torrent_unreachable",
              "torrent_invalid",
              "infohash_mismatch",
              "torrent_v2_only",
              "no_download_client",
              "download_failed",
              "no_video_file",
              "probe_failed",
              "fills_nothing",
              "transfer_failed",
              "import_failed",
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
    indexerId: faker.string.uuid(),
    infoHash: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    itemId: faker.string.uuid(),
    nodeId: faker.string.uuid(),
    ...(faker.datatype.boolean()
      ? { progress: { done: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) } }
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
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getGetGrabResponseMock = (
  overrideResponse: Partial<Extract<GrabsGrab, object>> = {},
): GrabsGrab => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  edition: faker.helpers.arrayElement(["standard"] as const),
  ...(faker.datatype.boolean()
    ? {
        failure: {
          code: faker.helpers.arrayElement([
            "indexer_unavailable",
            "torrent_unreachable",
            "torrent_invalid",
            "infohash_mismatch",
            "torrent_v2_only",
            "no_download_client",
            "download_failed",
            "no_video_file",
            "probe_failed",
            "fills_nothing",
            "transfer_failed",
            "import_failed",
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
  indexerId: faker.string.uuid(),
  infoHash: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  itemId: faker.string.uuid(),
  nodeId: faker.string.uuid(),
  ...(faker.datatype.boolean()
    ? { progress: { done: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) } }
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
      indexerId: faker.string.uuid(),
      itemId: faker.string.uuid(),
      outcome: faker.helpers.arrayElement([
        faker.helpers.arrayElement([
          "grabbed",
          "nothing_acceptable",
          "indexer_unavailable",
          "indexer_refused",
        ] as const),
        null,
      ]),
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
