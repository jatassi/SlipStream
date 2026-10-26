import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  DownloadclientsActed,
  DownloadclientsClientsList,
  DownloadclientsSettings,
  DownloadclientsThresholds,
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentList,
} from "../../model";

export const getListDownloadClientsResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsClientsList, object>> = {},
): DownloadclientsClientsList => ({
  cards: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    ...(faker.datatype.boolean()
      ? {
          client: {
            ...(faker.datatype.boolean()
              ? { bound: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            libtorrent: faker.string.alpha({ length: { min: 10, max: 20 } }),
            listen: faker.string.alpha({ length: { min: 10, max: 20 } }),
            ...(faker.datatype.boolean()
              ? { listenError: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            peerIdPrefix: faker.string.alpha({ length: { min: 10, max: 20 } }),
            reach: faker.helpers.arrayElement(["reachable", "unproven"] as const),
            ...(faker.datatype.boolean()
              ? { reachedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
              : {}),
            userAgent: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    ...(faker.datatype.boolean()
      ? {
          diskGuard: {
            errorBelow: faker.number.int({ min: 0 }),
            free: faker.number.int({ min: 0 }),
            hint: faker.string.alpha({ length: { min: 10, max: 20 } }),
            sampledAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            volume: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    inStep: faker.datatype.boolean(),
    limits: {
      activeChecks: faker.number.int({ min: 1 }),
      activeDownloads: faker.number.int({ min: 0 }),
      connections: faker.number.int({ min: 1 }),
      dht: faker.datatype.boolean(),
      downloadLimit: faker.number.int({ min: 0 }),
      stalledAfterSeconds: faker.number.int({ min: 1 }),
      uploadLimit: faker.number.int({ min: 0 }),
      utp: faker.datatype.boolean(),
    },
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    torrents: faker.number.int({ min: 0 }),
    totals: {
      activeSeconds: faker.number.int({ min: 0 }),
      downloaded: faker.number.int({ min: 0 }),
      seedingSeconds: faker.number.int({ min: 0 }),
      uploaded: faker.number.int({ min: 0 }),
    },
  })),
  thresholds: {
    faultStall: {
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
    metadataWait: {
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
    unregisteredGap: {
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
  },
  ...overrideResponse,
});

export const getSetDownloadClientThresholdsResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsThresholds, object>> = {},
): DownloadclientsThresholds => ({
  faultStall: {
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
  metadataWait: {
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
  unregisteredGap: {
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
  dht: {
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
  dht: {
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

export const getListTorrentsResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsTorrentList, object>> = {},
): DownloadclientsTorrentList => ({
  counts: {
    byNode: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({ nodeId: faker.string.uuid(), torrents: faker.number.int({ min: 0 }) }),
    ),
    byState: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
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
        torrents: faker.number.int({ min: 0 }),
      }),
    ),
    byTab: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        tab: faker.helpers.arrayElement(["downloading", "seeding", "problems", "all"] as const),
        torrents: faker.number.int({ min: 0 }),
      }),
    ),
  },
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
          ] as const),
        }
      : {}),
    id: faker.string.uuid(),
    ...(faker.datatype.boolean()
      ? { lastKnown: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
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
    tabs: faker.helpers.arrayElements(["downloading", "seeding", "problems", "all"] as const),
    ...(faker.datatype.boolean()
      ? {
          tracker: {
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
    ...(faker.datatype.boolean()
      ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    uploaded: faker.number.int({ min: 0 }),
    ...(faker.datatype.boolean() ? { workId: faker.string.uuid() } : {}),
  })),
  ...overrideResponse,
});

export const getActOnTorrentsResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsActed, object>> = {},
): DownloadclientsActed => ({
  ...(faker.datatype.boolean()
    ? {
        preview: {
          cancels: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            grabId: faker.string.uuid(),
            release: faker.string.alpha({ length: { min: 10, max: 20 } }),
            workId: faker.string.uuid(),
          })),
          frees: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
          linked: faker.datatype.boolean(),
          stopSeeding: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.uuid()),
        },
      }
    : {}),
  results: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      done: faker.datatype.boolean(),
      id: faker.string.uuid(),
      ...(faker.datatype.boolean()
        ? {
            refusal: {
              ...(faker.datatype.boolean()
                ? { acceptsAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
                : {}),
              code: faker.helpers.arrayElement([
                "not_found",
                "removed",
                "importing",
                "not_stalled",
                "offline",
                "window",
                "not_running",
                "failed",
              ] as const),
              ...(faker.datatype.boolean() ? { workId: faker.string.uuid() } : {}),
            },
          }
        : {}),
    }),
  ),
  ...overrideResponse,
});

export const getGetTorrentResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsTorrentDetail, object>> = {},
): DownloadclientsTorrentDetail => ({
  activeSeconds: faker.number.int({ min: 0 }),
  addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...(faker.datatype.boolean()
    ? { announceAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  done: faker.number.int({ min: 0 }),
  downloaded: faker.number.int({ min: 0 }),
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
          kind: faker.helpers.arrayElement(["engine_error", "unregistered", "no_payload"] as const),
          sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    ...(faker.datatype.boolean() ? { done: faker.number.int({ min: 0 }) } : {}),
    index: faker.number.int({ min: 0 }),
    ...(faker.datatype.boolean() ? { itemId: faker.string.uuid() } : {}),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    size: faker.number.int({ min: 0 }),
  })),
  grabs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.string.uuid(),
    itemId: faker.string.uuid(),
    release: faker.string.alpha({ length: { min: 10, max: 20 } }),
    stage: faker.helpers.arrayElement(["downloading", "importing", "imported", "failed"] as const),
    workId: faker.string.uuid(),
  })),
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
        ] as const),
      }
    : {}),
  id: faker.string.uuid(),
  infoHash: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { lastKnown: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  live: faker.datatype.boolean(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  nodeId: faker.string.uuid(),
  ...(faker.datatype.boolean()
    ? { pending: faker.helpers.arrayElement(["pause", "resume"] as const) }
    : {}),
  pieceSize: faker.number.int({ min: 1 }),
  private: faker.datatype.boolean(),
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
    ? { saveFolder: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
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
  tabs: faker.helpers.arrayElements(["downloading", "seeding", "problems", "all"] as const),
  ...(faker.datatype.boolean()
    ? {
        tracker: {
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
  trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean()
        ? { acceptsAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...(faker.datatype.boolean()
        ? { answeredAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...(faker.datatype.boolean()
        ? { message: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean() ? { minIntervalSeconds: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean()
        ? { nextAnnounce: faker.date.past().toISOString().slice(0, 19) + "Z" }
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
      tier: faker.number.int({ min: 0 }),
      url: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...(faker.datatype.boolean()
    ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  uploaded: faker.number.int({ min: 0 }),
  ...(faker.datatype.boolean() ? { workId: faker.string.uuid() } : {}),
  ...overrideResponse,
});

export const getListDownloadClientsMockHandler = (
  overrideResponse?:
    | DownloadclientsClientsList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<DownloadclientsClientsList> | DownloadclientsClientsList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/download-clients",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListDownloadClientsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetDownloadClientThresholdsMockHandler = (
  overrideResponse?:
    | DownloadclientsThresholds
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<DownloadclientsThresholds> | DownloadclientsThresholds),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/download-clients/thresholds",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetDownloadClientThresholdsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

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

export const getListTorrentsMockHandler = (
  overrideResponse?:
    | DownloadclientsTorrentList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<DownloadclientsTorrentList> | DownloadclientsTorrentList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/torrents",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTorrentsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getActOnTorrentsMockHandler = (
  overrideResponse?:
    | DownloadclientsActed
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<DownloadclientsActed> | DownloadclientsActed),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/torrents/actions",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getActOnTorrentsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetTorrentMockHandler = (
  overrideResponse?:
    | DownloadclientsTorrentDetail
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<DownloadclientsTorrentDetail> | DownloadclientsTorrentDetail),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/torrents/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTorrentResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getDownloadclientsMock = () => [
  getListDownloadClientsMockHandler(),
  getSetDownloadClientThresholdsMockHandler(),
  getGetDownloadClientSettingsMockHandler(),
  getSetDownloadClientSettingsMockHandler(),
  getListTorrentsMockHandler(),
  getActOnTorrentsMockHandler(),
  getGetTorrentMockHandler(),
];
