import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  SettingAddressField,
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingFolderField,
  SettingPortField,
  SettingRateField,
  SettingSpaceField,
  SettingSwitchField,
  SettingWindowField,
  TrackersGoal,
  TrackersSettings,
  TrackersTracker,
  TrackersTrackerList,
} from "../../model";

export const getListTrackersResponseMock = (
  overrideResponse: Partial<Extract<TrackersTrackerList, object>> = {},
): TrackersTrackerList => ({
  trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      goal: faker.helpers.arrayElement([
        {
          action: faker.helpers.arrayElement([
            faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
            null,
          ]),
          combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
          ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
          seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
        },
        null,
      ]),
      hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      id: faker.string.uuid(),
      indexerIds: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
      keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
        faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      privacy: faker.helpers.arrayElement([
        "private",
        "semi-private",
        "public",
        "unknown",
      ] as const),
      rule: faker.helpers.arrayElement([
        {
          clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
          ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
          seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
          source: faker.helpers.arrayElement(["releases", "admin"] as const),
        },
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getGetDefaultGoalResponseMock = (
  overrideResponse: Partial<Extract<TrackersGoal, object>> = {},
): TrackersGoal => ({
  action: faker.helpers.arrayElement([
    faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
    null,
  ]),
  combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
  ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
  seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
  ...overrideResponse,
});

export const getSetDefaultGoalResponseMock = (
  overrideResponse: Partial<Extract<TrackersGoal, object>> = {},
): TrackersGoal => ({
  action: faker.helpers.arrayElement([
    faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
    null,
  ]),
  combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
  ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
  seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
  ...overrideResponse,
});

export const getGetSeedingSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
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

export const getGetSeedingSettingsResponseSettingCountFieldMock = (
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

export const getGetSeedingSettingsResponseSettingChoiceFieldMock = (
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

export const getGetSeedingSettingsResponseSettingWindowFieldMock = (
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

export const getGetSeedingSettingsResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
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

export const getGetSeedingSettingsResponseSettingSwitchFieldMock = (
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

export const getGetSeedingSettingsResponseSettingRateFieldMock = (
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

export const getGetSeedingSettingsResponseSettingAddressFieldMock = (
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

export const getGetSeedingSettingsResponseSettingFolderFieldMock = (
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
    purpose: faker.helpers.arrayElement(["downloads"] as const),
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

export const getGetSeedingSettingsResponseSettingSpaceFieldMock = (
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

export const getGetSeedingSettingsResponseMock = (
  overrideResponse: Partial<Extract<TrackersSettings, object>> = {},
): TrackersSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetSeedingSettingsResponseSettingDurationFieldMock() },
      { ...getGetSeedingSettingsResponseSettingCountFieldMock() },
      { ...getGetSeedingSettingsResponseSettingChoiceFieldMock() },
      { ...getGetSeedingSettingsResponseSettingWindowFieldMock() },
      { ...getGetSeedingSettingsResponseSettingPortFieldMock() },
      { ...getGetSeedingSettingsResponseSettingSwitchFieldMock() },
      { ...getGetSeedingSettingsResponseSettingRateFieldMock() },
      { ...getGetSeedingSettingsResponseSettingAddressFieldMock() },
      { ...getGetSeedingSettingsResponseSettingFolderFieldMock() },
      { ...getGetSeedingSettingsResponseSettingSpaceFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateSeedingSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
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

export const getUpdateSeedingSettingsResponseSettingCountFieldMock = (
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

export const getUpdateSeedingSettingsResponseSettingChoiceFieldMock = (
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

export const getUpdateSeedingSettingsResponseSettingWindowFieldMock = (
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

export const getUpdateSeedingSettingsResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
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

export const getUpdateSeedingSettingsResponseSettingSwitchFieldMock = (
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

export const getUpdateSeedingSettingsResponseSettingRateFieldMock = (
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

export const getUpdateSeedingSettingsResponseSettingAddressFieldMock = (
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

export const getUpdateSeedingSettingsResponseSettingFolderFieldMock = (
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
    purpose: faker.helpers.arrayElement(["downloads"] as const),
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

export const getUpdateSeedingSettingsResponseSettingSpaceFieldMock = (
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

export const getUpdateSeedingSettingsResponseMock = (
  overrideResponse: Partial<Extract<TrackersSettings, object>> = {},
): TrackersSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateSeedingSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingCountFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingPortFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingRateFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateSeedingSettingsResponseSettingSpaceFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getGetTrackerResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getRenameTrackerResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getClearTrackerGoalResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getSetTrackerGoalResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getMergeTrackerResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getClearTrackerRuleResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getSetTrackerRuleResponseMock = (
  overrideResponse: Partial<Extract<TrackersTracker, object>> = {},
): TrackersTracker => ({
  goal: faker.helpers.arrayElement([
    {
      action: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["pause", "remove", "remove-delete"] as const),
        null,
      ]),
      combine: faker.helpers.arrayElement(["first", "both", "forever"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  hosts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  indexerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  keys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public", "unknown"] as const),
  rule: faker.helpers.arrayElement([
    {
      clears: faker.helpers.arrayElement(["time", "ratio", "either", "both"] as const),
      ratio: faker.helpers.arrayElement([faker.number.float({ fractionDigits: 2 }), null]),
      seedSeconds: faker.helpers.arrayElement([faker.number.int(), null]),
      source: faker.helpers.arrayElement(["releases", "admin"] as const),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getListTrackersMockHandler = (
  overrideResponse?:
    | TrackersTrackerList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TrackersTrackerList> | TrackersTrackerList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/trackers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTrackersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetDefaultGoalMockHandler = (
  overrideResponse?:
    | TrackersGoal
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TrackersGoal> | TrackersGoal),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/trackers/default-goal",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetDefaultGoalResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetDefaultGoalMockHandler = (
  overrideResponse?:
    | TrackersGoal
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TrackersGoal> | TrackersGoal),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/trackers/default-goal",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetDefaultGoalResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetSeedingSettingsMockHandler = (
  overrideResponse?:
    | TrackersSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TrackersSettings> | TrackersSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/trackers/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSeedingSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateSeedingSettingsMockHandler = (
  overrideResponse?:
    | TrackersSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<TrackersSettings> | TrackersSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/trackers/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateSeedingSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRemoveTrackerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/trackers/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetTrackerMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/trackers/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTrackerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRenameTrackerMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/trackers/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRenameTrackerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getClearTrackerGoalMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.delete>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/trackers/:id/goal",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getClearTrackerGoalResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetTrackerGoalMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/trackers/:id/goal",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetTrackerGoalResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getMergeTrackerMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/trackers/:id/merge",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getMergeTrackerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getClearTrackerRuleMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.delete>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/trackers/:id/rule",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getClearTrackerRuleResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetTrackerRuleMockHandler = (
  overrideResponse?:
    | TrackersTracker
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TrackersTracker> | TrackersTracker),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/trackers/:id/rule",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetTrackerRuleResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getTrackersMock = () => [
  getListTrackersMockHandler(),
  getGetDefaultGoalMockHandler(),
  getSetDefaultGoalMockHandler(),
  getGetSeedingSettingsMockHandler(),
  getUpdateSeedingSettingsMockHandler(),
  getRemoveTrackerMockHandler(),
  getGetTrackerMockHandler(),
  getRenameTrackerMockHandler(),
  getClearTrackerGoalMockHandler(),
  getSetTrackerGoalMockHandler(),
  getMergeTrackerMockHandler(),
  getClearTrackerRuleMockHandler(),
  getSetTrackerRuleMockHandler(),
];
