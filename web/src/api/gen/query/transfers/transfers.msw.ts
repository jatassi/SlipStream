import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  TransfersSettings,
  TransfersSpeedList,
  TransfersTransfer,
  TransfersTransferList,
} from "../../model";

export const getListTransfersResponseMock = (
  overrideResponse: Partial<Extract<TransfersTransferList, object>> = {},
): TransfersTransferList => ({
  transfers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      destination: {
        kind: faker.helpers.arrayElement(["root"] as const),
        nodeId: faker.string.uuid(),
        nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
        rootId: faker.string.uuid(),
      },
      details: {
        lanes: faker.number.int({ min: 0 }),
        retried: faker.number.int({ min: 0 }),
        rttMs: faker.number.int({ min: 0 }),
      },
      estimateSeconds: faker.number.int({ min: 0 }),
      files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          placed: faker.datatype.boolean(),
          position: faker.number.int({ min: 0 }),
          size: faker.number.int({ min: 0 }),
          verified: faker.datatype.boolean(),
        }),
      ),
      id: faker.string.uuid(),
      job: faker.string.alpha({ length: { min: 10, max: 20 } }),
      jobName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      purpose: faker.helpers.arrayElement(["grab"] as const),
      rate: faker.number.int({ min: 0 }),
      ...(faker.datatype.boolean()
        ? { reason: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? {
            route: faker.helpers.arrayElement([
              "receiver_dials",
              "sender_dials",
              "node_link",
            ] as const),
          }
        : {}),
      source: {
        nodeId: faker.string.uuid(),
        nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      state: faker.helpers.arrayElement([
        "queued",
        "waiting",
        "needs_space",
        "copying",
        "failed",
      ] as const),
      total: faker.number.int({ min: 0 }),
      verified: faker.number.int({ min: 0 }),
      ...(faker.datatype.boolean()
        ? {
            wait: {
              kind: faker.helpers.arrayElement([
                "node_away",
                "node_other_build",
                "connecting",
                "no_route",
                "root_not_ready",
                "storage_draining",
                "source_checking",
                "source_not_ready",
              ] as const),
              nodeId: faker.string.uuid(),
            },
          }
        : {}),
    }),
  ),
  ...overrideResponse,
});

export const getGetTransferSettingsResponseMock = (
  overrideResponse: Partial<Extract<TransfersSettings, object>> = {},
): TransfersSettings => ({
  sendLimit: {
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

export const getSetTransferSettingsResponseMock = (
  overrideResponse: Partial<Extract<TransfersSettings, object>> = {},
): TransfersSettings => ({
  sendLimit: {
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

export const getListTransferSpeedsResponseMock = (
  overrideResponse: Partial<Extract<TransfersSpeedList, object>> = {},
): TransfersSpeedList => ({
  speeds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      destinationNodeId: faker.string.uuid(),
      rate: faker.number.int({ min: 0 }),
      rateAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      ...(faker.datatype.boolean()
        ? {
            route: faker.helpers.arrayElement([
              "receiver_dials",
              "sender_dials",
              "node_link",
            ] as const),
          }
        : {}),
      ...(faker.datatype.boolean()
        ? { routeAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      sourceNodeId: faker.string.uuid(),
    }),
  ),
  ...overrideResponse,
});

export const getGetTransferResponseMock = (
  overrideResponse: Partial<Extract<TransfersTransfer, object>> = {},
): TransfersTransfer => ({
  destination: {
    kind: faker.helpers.arrayElement(["root"] as const),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  details: {
    lanes: faker.number.int({ min: 0 }),
    retried: faker.number.int({ min: 0 }),
    rttMs: faker.number.int({ min: 0 }),
  },
  estimateSeconds: faker.number.int({ min: 0 }),
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    placed: faker.datatype.boolean(),
    position: faker.number.int({ min: 0 }),
    size: faker.number.int({ min: 0 }),
    verified: faker.datatype.boolean(),
  })),
  id: faker.string.uuid(),
  job: faker.string.alpha({ length: { min: 10, max: 20 } }),
  jobName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  purpose: faker.helpers.arrayElement(["grab"] as const),
  rate: faker.number.int({ min: 0 }),
  ...(faker.datatype.boolean()
    ? { reason: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...(faker.datatype.boolean()
    ? {
        route: faker.helpers.arrayElement(["receiver_dials", "sender_dials", "node_link"] as const),
      }
    : {}),
  source: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  state: faker.helpers.arrayElement([
    "queued",
    "waiting",
    "needs_space",
    "copying",
    "failed",
  ] as const),
  total: faker.number.int({ min: 0 }),
  verified: faker.number.int({ min: 0 }),
  ...(faker.datatype.boolean()
    ? {
        wait: {
          kind: faker.helpers.arrayElement([
            "node_away",
            "node_other_build",
            "connecting",
            "no_route",
            "root_not_ready",
            "storage_draining",
            "source_checking",
            "source_not_ready",
          ] as const),
          nodeId: faker.string.uuid(),
        },
      }
    : {}),
  ...overrideResponse,
});

export const getListTransfersMockHandler = (
  overrideResponse?:
    | TransfersTransferList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TransfersTransferList> | TransfersTransferList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/transfers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTransfersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetTransferSettingsMockHandler = (
  overrideResponse?:
    | TransfersSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TransfersSettings> | TransfersSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/transfers/nodes/:nodeId/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTransferSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetTransferSettingsMockHandler = (
  overrideResponse?:
    | TransfersSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<TransfersSettings> | TransfersSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/transfers/nodes/:nodeId/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetTransferSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListTransferSpeedsMockHandler = (
  overrideResponse?:
    | TransfersSpeedList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TransfersSpeedList> | TransfersSpeedList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/transfers/speeds",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTransferSpeedsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetTransferMockHandler = (
  overrideResponse?:
    | TransfersTransfer
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TransfersTransfer> | TransfersTransfer),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/transfers/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTransferResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getTransfersMock = () => [
  getListTransfersMockHandler(),
  getGetTransferSettingsMockHandler(),
  getSetTransferSettingsMockHandler(),
  getListTransferSpeedsMockHandler(),
  getGetTransferMockHandler(),
];
