import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { NodesJoinCode, NodesNode, NodesNodeList, NodesSettings } from "../../model";

export const getListNodesResponseMock = (
  overrideResponse: Partial<Extract<NodesNodeList, object>> = {},
): NodesNodeList => ({
  nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? {
          facts: {
            appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
            assigned: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            restarting: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            running: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          },
        }
      : {}),
    id: faker.string.uuid(),
    intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    link: {
      ...(faker.datatype.boolean()
        ? {
            build: {
              arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
              installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
              os: faker.string.alpha({ length: { min: 10, max: 20 } }),
              version: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      since: faker.date.past().toISOString().slice(0, 19) + "Z",
      state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
    },
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    roles: faker.helpers.arrayElements([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    self: faker.datatype.boolean(),
  })),
  ...overrideResponse,
});

export const getCreateJoinCodeResponseMock = (
  overrideResponse: Partial<Extract<NodesJoinCode, object>> = {},
): NodesJoinCode => ({
  code: faker.string.alpha({ length: { min: 10, max: 20 } }),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getGetNodeSettingsResponseMock = (
  overrideResponse: Partial<Extract<NodesSettings, object>> = {},
): NodesSettings => ({
  joinCodeLifetime: {
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

export const getUpdateNodeSettingsResponseMock = (
  overrideResponse: Partial<Extract<NodesSettings, object>> = {},
): NodesSettings => ({
  joinCodeLifetime: {
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

export const getGetNodeResponseMock = (
  overrideResponse: Partial<Extract<NodesNode, object>> = {},
): NodesNode => ({
  addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...(faker.datatype.boolean()
    ? {
        facts: {
          appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
          assigned: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          restarting: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          running: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        },
      }
    : {}),
  id: faker.string.uuid(),
  intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  link: {
    ...(faker.datatype.boolean()
      ? {
          build: {
            arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
            installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            os: faker.string.alpha({ length: { min: 10, max: 20 } }),
            version: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
  },
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  roles: faker.helpers.arrayElements([
    "searching",
    "downloading",
    "storage",
    "converting",
    "subtitles",
    "metadata",
    "media-server-link",
    "notifying",
    "relaying",
    "serving",
  ] as const),
  self: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getSetNodeRolesResponseMock = (
  overrideResponse: Partial<Extract<NodesNode, object>> = {},
): NodesNode => ({
  addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...(faker.datatype.boolean()
    ? {
        facts: {
          appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
          assigned: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          restarting: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          running: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        },
      }
    : {}),
  id: faker.string.uuid(),
  intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  link: {
    ...(faker.datatype.boolean()
      ? {
          build: {
            arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
            installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            os: faker.string.alpha({ length: { min: 10, max: 20 } }),
            version: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
  },
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  roles: faker.helpers.arrayElements([
    "searching",
    "downloading",
    "storage",
    "converting",
    "subtitles",
    "metadata",
    "media-server-link",
    "notifying",
    "relaying",
    "serving",
  ] as const),
  self: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getListNodesMockHandler = (
  overrideResponse?:
    | NodesNodeList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesNodeList> | NodesNodeList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListNodesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateJoinCodeMockHandler = (
  overrideResponse?:
    | NodesJoinCode
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NodesJoinCode> | NodesJoinCode),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/nodes/join-codes",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateJoinCodeResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getGetNodeSettingsMockHandler = (
  overrideResponse?:
    | NodesSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesSettings> | NodesSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNodeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateNodeSettingsMockHandler = (
  overrideResponse?:
    | NodesSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<NodesSettings> | NodesSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/nodes/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateNodeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetNodeMockHandler = (
  overrideResponse?:
    | NodesNode
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<NodesNode> | NodesNode),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNodeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetNodeRolesMockHandler = (
  overrideResponse?:
    | NodesNode
    | ((info: Parameters<Parameters<typeof http.put>[1]>[0]) => Promise<NodesNode> | NodesNode),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/nodes/:id/roles",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetNodeRolesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getNodesMock = () => [
  getListNodesMockHandler(),
  getCreateJoinCodeMockHandler(),
  getGetNodeSettingsMockHandler(),
  getUpdateNodeSettingsMockHandler(),
  getGetNodeMockHandler(),
  getSetNodeRolesMockHandler(),
];
