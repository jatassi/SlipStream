import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  MovesBatchList,
  MovesBatchPage,
  MovesCanceledMoves,
  MovesMovePreview,
  MovesMoveSettings,
  MovesMoveUnit,
  MovesStartedMoves,
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

export const getListMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesBatchList, object>> = {},
): MovesBatchList => ({
  batches: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      canCancel: faker.datatype.boolean(),
      canMoveBack: faker.datatype.boolean(),
      canRetry: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { canceledAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      counts: {
        [faker.string.alphanumeric(5)]: faker.number.int(),
      },
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      ...(faker.datatype.boolean()
        ? { finishedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      id: faker.string.uuid(),
      origin: faker.helpers.arrayElement([
        "work",
        "bulk",
        "root",
        "drain",
        "client",
        "torrent",
      ] as const),
      total: faker.number.int({ min: 0 }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...(faker.datatype.boolean()
    ? { next: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...overrideResponse,
});

export const getStartMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesStartedMoves, object>> = {},
): MovesStartedMoves => ({
  batchId: faker.string.uuid(),
  dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      workId: faker.string.uuid(),
    }),
  ),
  started: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getPreviewMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesMovePreview, object>> = {},
): MovesMovePreview => ({
  button: faker.string.alpha({ length: { min: 10, max: 20 } }),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  homed: faker.number.int({ min: 0 }),
  movable: faker.number.int({ min: 0 }),
  recommendation: faker.string.alpha({ length: { min: 10, max: 20 } }),
  token: faker.string.alpha({ length: { min: 10, max: 20 } }),
  total: faker.number.int({ min: 0 }),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    bytes: faker.number.int({ min: 0 }),
    from: {
      ...(faker.datatype.boolean()
        ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["work", "torrent"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    reasons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        check: faker.helpers.arrayElement([
          "space",
          "names",
          "folder",
          "destination",
          "media_servers",
          "waits",
          "moving",
          "probe",
          "torrent",
          "hit_and_run",
        ] as const),
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        color: faker.helpers.arrayElement(["amber", "red"] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    to: {
      ...(faker.datatype.boolean()
        ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
    verdict: faker.helpers.arrayElement(["green", "amber", "red"] as const),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getGetMoveSettingsResponseSettingDurationFieldMock = (
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

export const getGetMoveSettingsResponseSettingCountFieldMock = (
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

export const getGetMoveSettingsResponseSettingChoiceFieldMock = (
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

export const getGetMoveSettingsResponseSettingWindowFieldMock = (
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

export const getGetMoveSettingsResponseSettingListenFieldMock = (
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

export const getGetMoveSettingsResponseSettingSwitchFieldMock = (
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

export const getGetMoveSettingsResponseSettingRateFieldMock = (
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

export const getGetMoveSettingsResponseSettingAddressFieldMock = (
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

export const getGetMoveSettingsResponseSettingFolderFieldMock = (
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

export const getGetMoveSettingsResponseSettingSpaceFieldMock = (
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

export const getGetMoveSettingsResponseSettingTemplateFieldMock = (
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

export const getGetMoveSettingsResponseSettingListFieldMock = (
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

export const getGetMoveSettingsResponseSettingSizeFieldMock = (
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

export const getGetMoveSettingsResponseSettingIPFieldMock = (
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

export const getGetMoveSettingsResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveSettings, object>> = {},
): MovesMoveSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetMoveSettingsResponseSettingDurationFieldMock() },
      { ...getGetMoveSettingsResponseSettingCountFieldMock() },
      { ...getGetMoveSettingsResponseSettingChoiceFieldMock() },
      { ...getGetMoveSettingsResponseSettingWindowFieldMock() },
      { ...getGetMoveSettingsResponseSettingListenFieldMock() },
      { ...getGetMoveSettingsResponseSettingSwitchFieldMock() },
      { ...getGetMoveSettingsResponseSettingRateFieldMock() },
      { ...getGetMoveSettingsResponseSettingAddressFieldMock() },
      { ...getGetMoveSettingsResponseSettingFolderFieldMock() },
      { ...getGetMoveSettingsResponseSettingSpaceFieldMock() },
      { ...getGetMoveSettingsResponseSettingTemplateFieldMock() },
      { ...getGetMoveSettingsResponseSettingListFieldMock() },
      { ...getGetMoveSettingsResponseSettingSizeFieldMock() },
      { ...getGetMoveSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getPatchMoveSettingsResponseSettingDurationFieldMock = (
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

export const getPatchMoveSettingsResponseSettingCountFieldMock = (
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

export const getPatchMoveSettingsResponseSettingChoiceFieldMock = (
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

export const getPatchMoveSettingsResponseSettingWindowFieldMock = (
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

export const getPatchMoveSettingsResponseSettingListenFieldMock = (
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

export const getPatchMoveSettingsResponseSettingSwitchFieldMock = (
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

export const getPatchMoveSettingsResponseSettingRateFieldMock = (
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

export const getPatchMoveSettingsResponseSettingAddressFieldMock = (
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

export const getPatchMoveSettingsResponseSettingFolderFieldMock = (
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

export const getPatchMoveSettingsResponseSettingSpaceFieldMock = (
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

export const getPatchMoveSettingsResponseSettingTemplateFieldMock = (
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

export const getPatchMoveSettingsResponseSettingListFieldMock = (
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

export const getPatchMoveSettingsResponseSettingSizeFieldMock = (
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

export const getPatchMoveSettingsResponseSettingIPFieldMock = (
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

export const getPatchMoveSettingsResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveSettings, object>> = {},
): MovesMoveSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getPatchMoveSettingsResponseSettingDurationFieldMock() },
      { ...getPatchMoveSettingsResponseSettingCountFieldMock() },
      { ...getPatchMoveSettingsResponseSettingChoiceFieldMock() },
      { ...getPatchMoveSettingsResponseSettingWindowFieldMock() },
      { ...getPatchMoveSettingsResponseSettingListenFieldMock() },
      { ...getPatchMoveSettingsResponseSettingSwitchFieldMock() },
      { ...getPatchMoveSettingsResponseSettingRateFieldMock() },
      { ...getPatchMoveSettingsResponseSettingAddressFieldMock() },
      { ...getPatchMoveSettingsResponseSettingFolderFieldMock() },
      { ...getPatchMoveSettingsResponseSettingSpaceFieldMock() },
      { ...getPatchMoveSettingsResponseSettingTemplateFieldMock() },
      { ...getPatchMoveSettingsResponseSettingListFieldMock() },
      { ...getPatchMoveSettingsResponseSettingSizeFieldMock() },
      { ...getPatchMoveSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getCancelMoveUnitResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveUnit, object>> = {},
): MovesMoveUnit => ({
  batchId: faker.string.uuid(),
  bytes: faker.number.int({ min: 0 }),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement([
        "source_gone",
        "source_failed",
        "damaged",
        "source_written_off",
        "destination_gone",
        "destination_check_failed",
        "switch_refused",
        "destination_unreachable",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  from: {
    ...(faker.datatype.boolean()
      ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["work", "torrent"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  phase: faker.helpers.arrayElement([
    "queued",
    "transfer",
    "place",
    "switch",
    "cleanup",
    "done",
    "failed",
    "canceled",
  ] as const),
  switchNow: faker.datatype.boolean(),
  to: {
    ...(faker.datatype.boolean()
      ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
  undone: faker.datatype.boolean(),
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
  ...(faker.datatype.boolean()
    ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getRetryMoveUnitResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveUnit, object>> = {},
): MovesMoveUnit => ({
  batchId: faker.string.uuid(),
  bytes: faker.number.int({ min: 0 }),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement([
        "source_gone",
        "source_failed",
        "damaged",
        "source_written_off",
        "destination_gone",
        "destination_check_failed",
        "switch_refused",
        "destination_unreachable",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  from: {
    ...(faker.datatype.boolean()
      ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["work", "torrent"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  phase: faker.helpers.arrayElement([
    "queued",
    "transfer",
    "place",
    "switch",
    "cleanup",
    "done",
    "failed",
    "canceled",
  ] as const),
  switchNow: faker.datatype.boolean(),
  to: {
    ...(faker.datatype.boolean()
      ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
  undone: faker.datatype.boolean(),
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
  ...(faker.datatype.boolean()
    ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getSwitchMoveUnitNowResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveUnit, object>> = {},
): MovesMoveUnit => ({
  batchId: faker.string.uuid(),
  bytes: faker.number.int({ min: 0 }),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement([
        "source_gone",
        "source_failed",
        "damaged",
        "source_written_off",
        "destination_gone",
        "destination_check_failed",
        "switch_refused",
        "destination_unreachable",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  from: {
    ...(faker.datatype.boolean()
      ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["work", "torrent"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  phase: faker.helpers.arrayElement([
    "queued",
    "transfer",
    "place",
    "switch",
    "cleanup",
    "done",
    "failed",
    "canceled",
  ] as const),
  switchNow: faker.datatype.boolean(),
  to: {
    ...(faker.datatype.boolean()
      ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
  undone: faker.datatype.boolean(),
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
  ...(faker.datatype.boolean()
    ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getGetMoveResponseMock = (
  overrideResponse: Partial<Extract<MovesBatchPage, object>> = {},
): MovesBatchPage => ({
  batch: {
    canCancel: faker.datatype.boolean(),
    canMoveBack: faker.datatype.boolean(),
    canRetry: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { canceledAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    counts: {
      [faker.string.alphanumeric(5)]: faker.number.int(),
    },
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    ...(faker.datatype.boolean()
      ? { finishedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    id: faker.string.uuid(),
    origin: faker.helpers.arrayElement([
      "work",
      "bulk",
      "root",
      "drain",
      "client",
      "torrent",
    ] as const),
    total: faker.number.int({ min: 0 }),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean() ? { next: faker.number.int() } : {}),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    batchId: faker.string.uuid(),
    bytes: faker.number.int({ min: 0 }),
    contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
    failure: faker.helpers.arrayElement([
      {
        code: faker.helpers.arrayElement([
          "source_gone",
          "source_failed",
          "damaged",
          "source_written_off",
          "destination_gone",
          "destination_check_failed",
          "switch_refused",
          "destination_unreachable",
        ] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    from: {
      ...(faker.datatype.boolean()
        ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    id: faker.string.uuid(),
    kind: faker.helpers.arrayElement(["work", "torrent"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    phase: faker.helpers.arrayElement([
      "queued",
      "transfer",
      "place",
      "switch",
      "cleanup",
      "done",
      "failed",
      "canceled",
    ] as const),
    switchNow: faker.datatype.boolean(),
    to: {
      ...(faker.datatype.boolean()
        ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
    undone: faker.datatype.boolean(),
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
    ...(faker.datatype.boolean()
      ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getPreviewMoveBackResponseMock = (
  overrideResponse: Partial<Extract<MovesMovePreview, object>> = {},
): MovesMovePreview => ({
  button: faker.string.alpha({ length: { min: 10, max: 20 } }),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  homed: faker.number.int({ min: 0 }),
  movable: faker.number.int({ min: 0 }),
  recommendation: faker.string.alpha({ length: { min: 10, max: 20 } }),
  token: faker.string.alpha({ length: { min: 10, max: 20 } }),
  total: faker.number.int({ min: 0 }),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    bytes: faker.number.int({ min: 0 }),
    from: {
      ...(faker.datatype.boolean()
        ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["work", "torrent"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    reasons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        check: faker.helpers.arrayElement([
          "space",
          "names",
          "folder",
          "destination",
          "media_servers",
          "waits",
          "moving",
          "probe",
          "torrent",
          "hit_and_run",
        ] as const),
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        color: faker.helpers.arrayElement(["amber", "red"] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    to: {
      ...(faker.datatype.boolean()
        ? { client: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    ...(faker.datatype.boolean() ? { torrentId: faker.string.uuid() } : {}),
    verdict: faker.helpers.arrayElement(["green", "amber", "red"] as const),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getCancelMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesCanceledMoves, object>> = {},
): MovesCanceledMoves => ({
  canceled: faker.number.int({ min: 0 }),
  switched: faker.number.int({ min: 0 }),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListMovesMockHandler = (
  overrideResponse?:
    | MovesBatchList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesBatchList> | MovesBatchList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMovesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartMovesMockHandler = (
  overrideResponse?:
    | MovesStartedMoves
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesStartedMoves> | MovesStartedMoves),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartMovesResponseMock(),
        { status: 202 },
      );
    },
    options,
  );
};

export const getPreviewMovesMockHandler = (
  overrideResponse?:
    | MovesMovePreview
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMovePreview> | MovesMovePreview),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/preview",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPreviewMovesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetMoveSettingsMockHandler = (
  overrideResponse?:
    | MovesMoveSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesMoveSettings> | MovesMoveSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetMoveSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPatchMoveSettingsMockHandler = (
  overrideResponse?:
    | MovesMoveSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<MovesMoveSettings> | MovesMoveSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/moves/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPatchMoveSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelMoveUnitMockHandler = (
  overrideResponse?:
    | MovesMoveUnit
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMoveUnit> | MovesMoveUnit),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/units/:unitId/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelMoveUnitResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRetryMoveUnitMockHandler = (
  overrideResponse?:
    | MovesMoveUnit
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMoveUnit> | MovesMoveUnit),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/units/:unitId/retry",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRetryMoveUnitResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSwitchMoveUnitNowMockHandler = (
  overrideResponse?:
    | MovesMoveUnit
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMoveUnit> | MovesMoveUnit),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/units/:unitId/switch-now",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSwitchMoveUnitNowResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetMoveMockHandler = (
  overrideResponse?:
    | MovesBatchPage
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesBatchPage> | MovesBatchPage),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves/:batchId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetMoveResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPreviewMoveBackMockHandler = (
  overrideResponse?:
    | MovesMovePreview
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesMovePreview> | MovesMovePreview),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves/:batchId/back",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPreviewMoveBackResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelMovesMockHandler = (
  overrideResponse?:
    | MovesCanceledMoves
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesCanceledMoves> | MovesCanceledMoves),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/:batchId/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelMovesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getMovesMock = () => [
  getListMovesMockHandler(),
  getStartMovesMockHandler(),
  getPreviewMovesMockHandler(),
  getGetMoveSettingsMockHandler(),
  getPatchMoveSettingsMockHandler(),
  getCancelMoveUnitMockHandler(),
  getRetryMoveUnitMockHandler(),
  getSwitchMoveUnitNowMockHandler(),
  getGetMoveMockHandler(),
  getPreviewMoveBackMockHandler(),
  getCancelMovesMockHandler(),
];
