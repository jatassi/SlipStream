import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { DownloadclientsSettings } from "../../model";

export const getGetDownloadClientSettingsResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsSettings, object>> = {},
): DownloadclientsSettings => ({
  activeChecks: {
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
  activeDownloads: {
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
  connections: {
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
  downloadLimit: {
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
  stalledAfter: {
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
  uploadLimit: {
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
  utp: {
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

export const getSetDownloadClientSettingsResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsSettings, object>> = {},
): DownloadclientsSettings => ({
  activeChecks: {
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
  activeDownloads: {
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
  connections: {
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
  downloadLimit: {
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
  stalledAfter: {
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
  uploadLimit: {
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
  utp: {
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

export const getGetDownloadClientSettingsMockHandler = (
  overrideResponse?:
    | DownloadclientsSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<DownloadclientsSettings> | DownloadclientsSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/download-clients/:nodeId/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetDownloadClientSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetDownloadClientSettingsMockHandler = (
  overrideResponse?:
    | DownloadclientsSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<DownloadclientsSettings> | DownloadclientsSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/download-clients/:nodeId/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetDownloadClientSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getDownloadclientsMock = () => [
  getGetDownloadClientSettingsMockHandler(),
  getSetDownloadClientSettingsMockHandler(),
];
