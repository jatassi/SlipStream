import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  DownloadclientsActed,
  DownloadclientsClientsList,
  DownloadclientsExternalClient,
  DownloadclientsSettings,
  DownloadclientsTestReport,
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
            hint: faker.string.alpha({ length: { min: 10, max: 20 } }),
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    ...(faker.datatype.boolean()
      ? {
          downloads: {
            free: faker.number.int({ min: 0 }),
            mount: faker.string.alpha({ length: { min: 10, max: 20 } }),
            name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            sampledAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            total: faker.number.int({ min: 0 }),
          },
        }
      : {}),
    externals: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        actions: faker.helpers.arrayElements(["pause", "resume", "recheck", "announce"] as const),
        address: faker.string.alpha({ length: { min: 10, max: 20 } }),
        apiKeySaved: faker.datatype.boolean(),
        away: faker.datatype.boolean(),
        checkEvery: {
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
        clientFolders: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => ({
          client: faker.string.alpha({ length: { min: 1, max: 4096 } }),
          node: faker.string.alpha({ length: { min: 1, max: 4096 } }),
        })),
        daemon: faker.helpers.arrayElement([
          {
            host: faker.string.alpha({ length: { min: 1, max: 253 } }),
            port: faker.number.int({ min: 1, max: 65535 }),
          },
          null,
        ]),
        folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
        id: faker.string.uuid(),
        importDelay: {
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
        kind: faker.helpers.arrayElement([
          "qbittorrent",
          "transmission",
          "deluge",
          "rtorrent",
        ] as const),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        nodeId: faker.string.uuid(),
        on: faker.datatype.boolean(),
        passwordSaved: faker.datatype.boolean(),
        problem: faker.helpers.arrayElement([
          {
            cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
            fields: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          null,
        ]),
        productVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
        reach: faker.helpers.arrayElement([
          "responding",
          "not_responding",
          "sign_in_refused",
          "needs_attention",
          "off",
          "node_offline",
          "unchecked",
        ] as const),
        settingsVersion: faker.number.int({ min: 1 }),
        since: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        torrents: faker.number.int({ min: 0 }),
        trackers: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => ({
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          stored: faker.datatype.boolean(),
        })),
        username: faker.string.alpha({ length: { min: 10, max: 20 } }),
        webUi: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
      }),
    ),
    grabsTo: faker.helpers.arrayElement([faker.string.uuid(), null]),
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
    trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        stored: faker.datatype.boolean(),
      }),
    ),
  })),
  kinds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    actions: faker.helpers.arrayElements(["pause", "resume", "recheck", "announce"] as const),
    apiKey: faker.datatype.boolean(),
    daemon: faker.datatype.boolean(),
    deletesData: faker.datatype.boolean(),
    floor: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement([
      "qbittorrent",
      "transmission",
      "deluge",
      "rtorrent",
    ] as const),
    label: faker.helpers.arrayElement(["any", "no_comma", "lower_word"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    password: faker.helpers.arrayElement(["none", "optional", "required"] as const),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    username: faker.helpers.arrayElement(["none", "optional", "required"] as const),
  })),
  orphans: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      actions: faker.helpers.arrayElements(["pause", "resume", "recheck", "announce"] as const),
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      apiKeySaved: faker.datatype.boolean(),
      away: faker.datatype.boolean(),
      checkEvery: {
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
      clientFolders: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => ({
        client: faker.string.alpha({ length: { min: 1, max: 4096 } }),
        node: faker.string.alpha({ length: { min: 1, max: 4096 } }),
      })),
      daemon: faker.helpers.arrayElement([
        {
          host: faker.string.alpha({ length: { min: 1, max: 253 } }),
          port: faker.number.int({ min: 1, max: 65535 }),
        },
        null,
      ]),
      folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.uuid(),
      importDelay: {
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
      kind: faker.helpers.arrayElement([
        "qbittorrent",
        "transmission",
        "deluge",
        "rtorrent",
      ] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodeId: faker.string.uuid(),
      on: faker.datatype.boolean(),
      passwordSaved: faker.datatype.boolean(),
      problem: faker.helpers.arrayElement([
        {
          cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
          fields: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
        null,
      ]),
      productVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
      reach: faker.helpers.arrayElement([
        "responding",
        "not_responding",
        "sign_in_refused",
        "needs_attention",
        "off",
        "node_offline",
        "unchecked",
      ] as const),
      settingsVersion: faker.number.int({ min: 1 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      torrents: faker.number.int({ min: 0 }),
      trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          stored: faker.datatype.boolean(),
        }),
      ),
      username: faker.string.alpha({ length: { min: 10, max: 20 } }),
      webUi: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
    }),
  ),
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
  grabsTo: faker.helpers.arrayElement([faker.string.uuid(), null]),
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
  grabsTo: faker.helpers.arrayElement([faker.string.uuid(), null]),
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

export const getCreateDownloadClientResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsExternalClient, object>> = {},
): DownloadclientsExternalClient => ({
  actions: faker.helpers.arrayElements(["pause", "resume", "recheck", "announce"] as const),
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  apiKeySaved: faker.datatype.boolean(),
  away: faker.datatype.boolean(),
  checkEvery: {
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
  clientFolders: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      client: faker.string.alpha({ length: { min: 1, max: 4096 } }),
      node: faker.string.alpha({ length: { min: 1, max: 4096 } }),
    }),
  ),
  daemon: faker.helpers.arrayElement([
    {
      host: faker.string.alpha({ length: { min: 1, max: 253 } }),
      port: faker.number.int({ min: 1, max: 65535 }),
    },
    null,
  ]),
  folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  id: faker.string.uuid(),
  importDelay: {
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
  kind: faker.helpers.arrayElement(["qbittorrent", "transmission", "deluge", "rtorrent"] as const),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  nodeId: faker.string.uuid(),
  on: faker.datatype.boolean(),
  passwordSaved: faker.datatype.boolean(),
  problem: faker.helpers.arrayElement([
    {
      cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  productVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
  reach: faker.helpers.arrayElement([
    "responding",
    "not_responding",
    "sign_in_refused",
    "needs_attention",
    "off",
    "node_offline",
    "unchecked",
  ] as const),
  settingsVersion: faker.number.int({ min: 1 }),
  since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  torrents: faker.number.int({ min: 0 }),
  trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      stored: faker.datatype.boolean(),
    }),
  ),
  username: faker.string.alpha({ length: { min: 10, max: 20 } }),
  webUi: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  ...overrideResponse,
});

export const getTestDownloadClientDraftResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsTestReport, object>> = {},
): DownloadclientsTestReport => ({
  daemons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      host: faker.string.alpha({ length: { min: 1, max: 253 } }),
      port: faker.number.int({ min: 1, max: 65535 }),
    }),
  ),
  nodeFolder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  product: faker.string.alpha({ length: { min: 10, max: 20 } }),
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  warnings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getUpdateDownloadClientResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsExternalClient, object>> = {},
): DownloadclientsExternalClient => ({
  actions: faker.helpers.arrayElements(["pause", "resume", "recheck", "announce"] as const),
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  apiKeySaved: faker.datatype.boolean(),
  away: faker.datatype.boolean(),
  checkEvery: {
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
  clientFolders: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      client: faker.string.alpha({ length: { min: 1, max: 4096 } }),
      node: faker.string.alpha({ length: { min: 1, max: 4096 } }),
    }),
  ),
  daemon: faker.helpers.arrayElement([
    {
      host: faker.string.alpha({ length: { min: 1, max: 253 } }),
      port: faker.number.int({ min: 1, max: 65535 }),
    },
    null,
  ]),
  folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  id: faker.string.uuid(),
  importDelay: {
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
  kind: faker.helpers.arrayElement(["qbittorrent", "transmission", "deluge", "rtorrent"] as const),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  nodeId: faker.string.uuid(),
  on: faker.datatype.boolean(),
  passwordSaved: faker.datatype.boolean(),
  problem: faker.helpers.arrayElement([
    {
      cause: faker.string.alpha({ length: { min: 10, max: 20 } }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  productVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
  reach: faker.helpers.arrayElement([
    "responding",
    "not_responding",
    "sign_in_refused",
    "needs_attention",
    "off",
    "node_offline",
    "unchecked",
  ] as const),
  settingsVersion: faker.number.int({ min: 1 }),
  since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  torrents: faker.number.int({ min: 0 }),
  trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      stored: faker.datatype.boolean(),
    }),
  ),
  username: faker.string.alpha({ length: { min: 10, max: 20 } }),
  webUi: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  ...overrideResponse,
});

export const getTestDownloadClientResponseMock = (
  overrideResponse: Partial<Extract<DownloadclientsTestReport, object>> = {},
): DownloadclientsTestReport => ({
  daemons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      host: faker.string.alpha({ length: { min: 1, max: 253 } }),
      port: faker.number.int({ min: 1, max: 65535 }),
    }),
  ),
  nodeFolder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  product: faker.string.alpha({ length: { min: 10, max: 20 } }),
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  warnings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
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
        tab: faker.helpers.arrayElement([
          "downloading",
          "seeding",
          "problems",
          "goal-met",
          "all",
        ] as const),
        torrents: faker.number.int({ min: 0 }),
      }),
    ),
  },
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
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
          ? { action: faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const) }
          : {}),
        atRisk: faker.helpers.arrayElement(["no", "yes", "unknown"] as const),
        clears: faker.helpers.arrayElement(["none", "time", "ratio", "either", "both"] as const),
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
          ? { owes: { bytes: faker.number.int({ min: 0 }), seconds: faker.number.int({ min: 0 }) } }
          : {}),
        ratio: {
          ...(faker.datatype.boolean() ? { goal: faker.number.float({ fractionDigits: 2 }) } : {}),
          ...(faker.datatype.boolean() ? { rule: faker.number.float({ fractionDigits: 2 }) } : {}),
          value: faker.number.float({ min: 0, fractionDigits: 2 }),
        },
        state: faker.helpers.arrayElement(["none", "owing", "met", "missed", "released"] as const),
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
      { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      null,
    ]),
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
          hitAndRun: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            clears: faker.helpers.arrayElement([
              "none",
              "time",
              "ratio",
              "either",
              "both",
            ] as const),
            ...(faker.datatype.boolean()
              ? {
                  owes: {
                    bytes: faker.number.int({ min: 0 }),
                    seconds: faker.number.int({ min: 0 }),
                  },
                }
              : {}),
            torrentId: faker.string.uuid(),
            tracker: faker.helpers.arrayElement([
              {
                id: faker.string.uuid(),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
          })),
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
              ...(faker.datatype.boolean()
                ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              code: faker.helpers.arrayElement([
                "not_found",
                "removed",
                "importing",
                "not_held",
                "reconciling",
                "offline",
                "client_away",
                "client_off",
                "window",
                "not_running",
                "failed",
                "not_retryable",
              ] as const),
              ...(faker.datatype.boolean()
                ? { node: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
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
  ...(faker.datatype.boolean()
    ? { announceAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
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
  infoHash: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { lastKnown: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...(faker.datatype.boolean()
    ? { lastKnownCause: faker.helpers.arrayElement(["node", "client", "off"] as const) }
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
    ? {
        retry: {
          at: faker.date.past().toISOString().slice(0, 19) + "Z",
          ...(faker.datatype.boolean() ? { found: faker.number.int({ min: 0 }) } : {}),
        },
      }
    : {}),
  retryable: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? { saveFolder: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  seeding: faker.helpers.arrayElement([
    {
      ...(faker.datatype.boolean()
        ? { action: faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const) }
        : {}),
      atRisk: faker.helpers.arrayElement(["no", "yes", "unknown"] as const),
      clears: faker.helpers.arrayElement(["none", "time", "ratio", "either", "both"] as const),
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
        ? { owes: { bytes: faker.number.int({ min: 0 }), seconds: faker.number.int({ min: 0 }) } }
        : {}),
      ratio: {
        ...(faker.datatype.boolean() ? { goal: faker.number.float({ fractionDigits: 2 }) } : {}),
        ...(faker.datatype.boolean() ? { rule: faker.number.float({ fractionDigits: 2 }) } : {}),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
      },
      state: faker.helpers.arrayElement(["none", "owing", "met", "missed", "released"] as const),
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
    { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean()
        ? { acceptsAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...(faker.datatype.boolean()
        ? { answeredAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...(faker.datatype.boolean() ? { leechers: faker.number.int({ min: 0 }) } : {}),
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
      ...(faker.datatype.boolean() ? { seeders: faker.number.int({ min: 0 }) } : {}),
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

export const getCreateDownloadClientMockHandler = (
  overrideResponse?:
    | DownloadclientsExternalClient
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<DownloadclientsExternalClient> | DownloadclientsExternalClient),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/external-download-clients",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateDownloadClientResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getTestDownloadClientDraftMockHandler = (
  overrideResponse?:
    | DownloadclientsTestReport
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<DownloadclientsTestReport> | DownloadclientsTestReport),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/external-download-clients/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestDownloadClientDraftResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteDownloadClientMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/external-download-clients/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getUpdateDownloadClientMockHandler = (
  overrideResponse?:
    | DownloadclientsExternalClient
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<DownloadclientsExternalClient> | DownloadclientsExternalClient),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/external-download-clients/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateDownloadClientResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getTestDownloadClientMockHandler = (
  overrideResponse?:
    | DownloadclientsTestReport
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<DownloadclientsTestReport> | DownloadclientsTestReport),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/external-download-clients/:id/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestDownloadClientResponseMock(),
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
  getCreateDownloadClientMockHandler(),
  getTestDownloadClientDraftMockHandler(),
  getDeleteDownloadClientMockHandler(),
  getUpdateDownloadClientMockHandler(),
  getTestDownloadClientMockHandler(),
  getListTorrentsMockHandler(),
  getActOnTorrentsMockHandler(),
  getGetTorrentMockHandler(),
];
