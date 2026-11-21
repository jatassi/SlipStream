import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { ReconcilingPage, ReconcilingRehomed, ReconcilingReviewPage } from "../../model";

export const getGetReconcilingResponseMock = (
  overrideResponse: Partial<Extract<ReconcilingPage, object>> = {},
): ReconcilingPage => ({
  nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    adopted: faker.datatype.boolean(),
    continued: faker.datatype.boolean(),
    granted: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    held: faker.number.int({ min: 0 }),
    id: faker.string.uuid(),
    link: faker.helpers.arrayElement([
      "in-step",
      "reconnecting",
      "offline",
      "lapsed",
      "mismatched",
    ] as const),
    linkSince: faker.date.past().toISOString().slice(0, 19) + "Z",
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    refused: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    reported: faker.datatype.boolean(),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
  })),
  restore: faker.helpers.arrayElement([
    {
      backup: faker.string.alpha({ length: { min: 10, max: 20 } }),
      newMachine: faker.datatype.boolean(),
      rehome: faker.datatype.boolean(),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      takenAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    },
    null,
  ]),
  review: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      count: faker.number.int({ min: 0 }),
      found: faker.helpers.arrayElement(["not_looked", "no_files", "files"] as const),
      reason: faker.helpers.arrayElement(["not_held", "conflict"] as const),
    }),
  ),
  roots: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    continued: faker.datatype.boolean(),
    id: faker.string.uuid(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    saved: faker.datatype.boolean(),
    scanned: faker.datatype.boolean(),
  })),
  unknown: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      fingerprint: faker.string.alpha({ length: { min: 10, max: 20 } }),
      os: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getRehomeReconcilingResponseMock = (
  overrideResponse: Partial<Extract<ReconcilingRehomed, object>> = {},
): ReconcilingRehomed => ({
  roots: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    missing: faker.helpers.arrayElement([faker.number.int(), null]),
    refused: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    rootId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getListReconcilingTorrentsResponseMock = (
  overrideResponse: Partial<Extract<ReconcilingReviewPage, object>> = {},
): ReconcilingReviewPage => ({
  next: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  torrents: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      found: faker.helpers.arrayElement(["not_looked", "no_files", "files"] as const),
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodeId: faker.string.uuid(),
      readd: faker.helpers.arrayElement([faker.string.uuid(), null]),
      reason: faker.helpers.arrayElement(["not_held", "conflict"] as const),
      since: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  ...overrideResponse,
});

export const getGetReconcilingMockHandler = (
  overrideResponse?:
    | ReconcilingPage
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ReconcilingPage> | ReconcilingPage),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/reconciling",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetReconcilingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getFinishReconcilingMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reconciling/finish",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getContinueReconcilingNodeMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reconciling/nodes/:nodeId/continue",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getRehomeReconcilingMockHandler = (
  overrideResponse?:
    | ReconcilingRehomed
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ReconcilingRehomed> | ReconcilingRehomed),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reconciling/rehome",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRehomeReconcilingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getContinueReconcilingRootMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reconciling/roots/:rootId/continue",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListReconcilingTorrentsMockHandler = (
  overrideResponse?:
    | ReconcilingReviewPage
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ReconcilingReviewPage> | ReconcilingReviewPage),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/reconciling/torrents",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListReconcilingTorrentsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getForgetReconcilingTorrentsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reconciling/torrents/forget",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getReaddReconcilingTorrentsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reconciling/torrents/readd",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};
export const getReconcilingMock = () => [
  getGetReconcilingMockHandler(),
  getFinishReconcilingMockHandler(),
  getContinueReconcilingNodeMockHandler(),
  getRehomeReconcilingMockHandler(),
  getContinueReconcilingRootMockHandler(),
  getListReconcilingTorrentsMockHandler(),
  getForgetReconcilingTorrentsMockHandler(),
  getReaddReconcilingTorrentsMockHandler(),
];
