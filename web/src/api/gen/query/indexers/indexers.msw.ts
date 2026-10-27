import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  IndexersCatalogEntry,
  IndexersDefinitionForm,
  IndexersIndexer,
  IndexersIndexerCatalog,
  IndexersIndexerList,
  IndexersQueryList,
  IndexersReport,
  IndexersSettings,
  IndexersTracker,
  IndexersTrackerList,
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
} from "../../model";

export const getGetIndexerCatalogResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerCatalog, object>> = {},
): IndexersIndexerCatalog => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  addressIsDefault: faker.datatype.boolean(),
  entries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      categories: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.number.int()),
      description: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      language: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
      privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
      reason: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      replacedByCustom: faker.datatype.boolean(),
      version: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  nextRefreshAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  notice: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  refreshedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  ...overrideResponse,
});

export const getSetCatalogAddressResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerCatalog, object>> = {},
): IndexersIndexerCatalog => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  addressIsDefault: faker.datatype.boolean(),
  entries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      categories: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.number.int()),
      description: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      language: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
      privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
      reason: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      replacedByCustom: faker.datatype.boolean(),
      version: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  nextRefreshAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  notice: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  refreshedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  ...overrideResponse,
});

export const getAddCustomDefinitionResponseMock = (
  overrideResponse: Partial<Extract<IndexersCatalogEntry, object>> = {},
): IndexersCatalogEntry => ({
  categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.number.int(),
  ),
  description: faker.string.alpha({ length: { min: 10, max: 20 } }),
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  language: faker.string.alpha({ length: { min: 10, max: 20 } }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
  reason: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  replacedByCustom: faker.datatype.boolean(),
  version: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getGetCatalogDefinitionResponseMock = (
  overrideResponse: Partial<Extract<IndexersDefinitionForm, object>> = {},
): IndexersDefinitionForm => ({
  captcha: faker.datatype.boolean(),
  entry: {
    categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.number.int(),
    ),
    description: faker.string.alpha({ length: { min: 10, max: 20 } }),
    id: faker.string.alpha({ length: { min: 10, max: 20 } }),
    language: faker.string.alpha({ length: { min: 10, max: 20 } }),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
    privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
    reason: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    replacedByCustom: faker.datatype.boolean(),
    version: faker.date.past().toISOString().slice(0, 19) + "Z",
  },
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      default: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      help: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          value: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      sealed: faker.datatype.boolean(),
      type: faker.helpers.arrayElement([
        "text",
        "password",
        "checkbox",
        "select",
        "info",
        "info_cookie",
        "info_flaresolverr",
        "info_useragent",
        "info_category_8000",
        "cardigannCaptcha",
      ] as const),
    }),
  ),
  links: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  session: faker.datatype.boolean(),
  solver: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getGetIndexerSettingsResponseSettingDurationFieldMock = (
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

export const getGetIndexerSettingsResponseSettingCountFieldMock = (
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

export const getGetIndexerSettingsResponseSettingChoiceFieldMock = (
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

export const getGetIndexerSettingsResponseSettingWindowFieldMock = (
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

export const getGetIndexerSettingsResponseSettingPortFieldMock = (
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

export const getGetIndexerSettingsResponseSettingSwitchFieldMock = (
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

export const getGetIndexerSettingsResponseSettingRateFieldMock = (
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

export const getGetIndexerSettingsResponseSettingAddressFieldMock = (
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

export const getGetIndexerSettingsResponseSettingFolderFieldMock = (
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

export const getGetIndexerSettingsResponseSettingSpaceFieldMock = (
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

export const getGetIndexerSettingsResponseMock = (
  overrideResponse: Partial<Extract<IndexersSettings, object>> = {},
): IndexersSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetIndexerSettingsResponseSettingDurationFieldMock() },
      { ...getGetIndexerSettingsResponseSettingCountFieldMock() },
      { ...getGetIndexerSettingsResponseSettingChoiceFieldMock() },
      { ...getGetIndexerSettingsResponseSettingWindowFieldMock() },
      { ...getGetIndexerSettingsResponseSettingPortFieldMock() },
      { ...getGetIndexerSettingsResponseSettingSwitchFieldMock() },
      { ...getGetIndexerSettingsResponseSettingRateFieldMock() },
      { ...getGetIndexerSettingsResponseSettingAddressFieldMock() },
      { ...getGetIndexerSettingsResponseSettingFolderFieldMock() },
      { ...getGetIndexerSettingsResponseSettingSpaceFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateIndexerSettingsResponseSettingDurationFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingCountFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingChoiceFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingWindowFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingPortFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingSwitchFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingRateFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingAddressFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingFolderFieldMock = (
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

export const getUpdateIndexerSettingsResponseSettingSpaceFieldMock = (
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

export const getUpdateIndexerSettingsResponseMock = (
  overrideResponse: Partial<Extract<IndexersSettings, object>> = {},
): IndexersSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateIndexerSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingCountFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingPortFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingRateFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateIndexerSettingsResponseSettingSpaceFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getListIndexersResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerList, object>> = {},
): IndexersIndexerList => ({
  indexers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      allowFailover: faker.datatype.boolean(),
      allowNoSize: faker.datatype.boolean(),
      automaticSearch: faker.datatype.boolean(),
      categories: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.number.int(),
        ),
        null,
      ]),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      feedHost: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      fix: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
      grabs: {
        count: faker.number.int({ min: 0 }),
        freeAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        per: faker.helpers.arrayElement(["hour", "day"] as const),
        used: faker.number.int({ min: 0 }),
      },
      homeNodeId: faker.string.uuid(),
      id: faker.string.uuid(),
      kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
      media: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      on: faker.datatype.boolean(),
      priority: faker.number.int({ min: 1, max: 50 }),
      revision: faker.number.int({ min: 1 }),
      rss: faker.datatype.boolean(),
      searchModes: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => ({
        mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
        params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      })),
      searches: {
        count: faker.number.int({ min: 0 }),
        freeAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        per: faker.helpers.arrayElement(["hour", "day"] as const),
        used: faker.number.int({ min: 0 }),
      },
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
      session: faker.helpers.arrayElement([
        {
          earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          nodeId: faker.string.uuid(),
          pasted: faker.datatype.boolean(),
        },
        null,
      ]),
      signsIn: faker.datatype.boolean(),
      status: {
        cause: faker.helpers.arrayElement([
          faker.helpers.arrayElement([
            "credentials_rejected",
            "captcha_needed",
            "challenge_unsolved",
            "cookie_expired",
            "definition_unloadable",
            "account_refused",
            "request_rejected",
          ] as const),
          null,
        ]),
        lastFailure: faker.helpers.arrayElement([
          {
            at: faker.date.past().toISOString().slice(0, 19) + "Z",
            text: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          null,
        ]),
        nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        rung: faker.number.int({ min: 0 }),
        since: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        solverTried: faker.datatype.boolean(),
        state: faker.helpers.arrayElement([
          "ok",
          "limited",
          "failing",
          "stopped",
          "home_only",
        ] as const),
        successAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        text: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        until: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      },
      trackerIds: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
      waits: faker.helpers.arrayElement([
        {
          cause: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          kind: faker.helpers.arrayElement([
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
          lapsed: faker.datatype.boolean(),
          nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          releases: faker.number.int({ min: 0 }),
          since: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          text: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          until: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
        },
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getCreateIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  allowFailover: faker.datatype.boolean(),
  allowNoSize: faker.datatype.boolean(),
  automaticSearch: faker.datatype.boolean(),
  categories: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.number.int(),
    ),
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  feedHost: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  fix: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  grabs: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
  media: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  on: faker.datatype.boolean(),
  priority: faker.number.int({ min: 1, max: 50 }),
  revision: faker.number.int({ min: 1 }),
  rss: faker.datatype.boolean(),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  searches: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
  session: faker.helpers.arrayElement([
    {
      earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      nodeId: faker.string.uuid(),
      pasted: faker.datatype.boolean(),
    },
    null,
  ]),
  signsIn: faker.datatype.boolean(),
  status: {
    cause: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "credentials_rejected",
        "captcha_needed",
        "challenge_unsolved",
        "cookie_expired",
        "definition_unloadable",
        "account_refused",
        "request_rejected",
      ] as const),
      null,
    ]),
    lastFailure: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    rung: faker.number.int({ min: 0 }),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    solverTried: faker.datatype.boolean(),
    state: faker.helpers.arrayElement([
      "ok",
      "limited",
      "failing",
      "stopped",
      "home_only",
    ] as const),
    successAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    text: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  trackerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  waits: faker.helpers.arrayElement([
    {
      cause: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      kind: faker.helpers.arrayElement([
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
      lapsed: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      releases: faker.number.int({ min: 0 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      text: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getTestIndexerDraftResponseMock = (
  overrideResponse: Partial<Extract<IndexersReport, object>> = {},
): IndexersReport => ({
  items: faker.number.int({ min: 0 }),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  sources: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getGetIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  allowFailover: faker.datatype.boolean(),
  allowNoSize: faker.datatype.boolean(),
  automaticSearch: faker.datatype.boolean(),
  categories: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.number.int(),
    ),
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  feedHost: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  fix: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  grabs: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
  media: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  on: faker.datatype.boolean(),
  priority: faker.number.int({ min: 1, max: 50 }),
  revision: faker.number.int({ min: 1 }),
  rss: faker.datatype.boolean(),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  searches: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
  session: faker.helpers.arrayElement([
    {
      earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      nodeId: faker.string.uuid(),
      pasted: faker.datatype.boolean(),
    },
    null,
  ]),
  signsIn: faker.datatype.boolean(),
  status: {
    cause: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "credentials_rejected",
        "captcha_needed",
        "challenge_unsolved",
        "cookie_expired",
        "definition_unloadable",
        "account_refused",
        "request_rejected",
      ] as const),
      null,
    ]),
    lastFailure: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    rung: faker.number.int({ min: 0 }),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    solverTried: faker.datatype.boolean(),
    state: faker.helpers.arrayElement([
      "ok",
      "limited",
      "failing",
      "stopped",
      "home_only",
    ] as const),
    successAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    text: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  trackerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  waits: faker.helpers.arrayElement([
    {
      cause: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      kind: faker.helpers.arrayElement([
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
      lapsed: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      releases: faker.number.int({ min: 0 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      text: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getUpdateIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  allowFailover: faker.datatype.boolean(),
  allowNoSize: faker.datatype.boolean(),
  automaticSearch: faker.datatype.boolean(),
  categories: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.number.int(),
    ),
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  feedHost: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  fix: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  grabs: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
  media: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  on: faker.datatype.boolean(),
  priority: faker.number.int({ min: 1, max: 50 }),
  revision: faker.number.int({ min: 1 }),
  rss: faker.datatype.boolean(),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  searches: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
  session: faker.helpers.arrayElement([
    {
      earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      nodeId: faker.string.uuid(),
      pasted: faker.datatype.boolean(),
    },
    null,
  ]),
  signsIn: faker.datatype.boolean(),
  status: {
    cause: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "credentials_rejected",
        "captcha_needed",
        "challenge_unsolved",
        "cookie_expired",
        "definition_unloadable",
        "account_refused",
        "request_rejected",
      ] as const),
      null,
    ]),
    lastFailure: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    rung: faker.number.int({ min: 0 }),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    solverTried: faker.datatype.boolean(),
    state: faker.helpers.arrayElement([
      "ok",
      "limited",
      "failing",
      "stopped",
      "home_only",
    ] as const),
    successAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    text: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  trackerIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  waits: faker.helpers.arrayElement([
    {
      cause: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      kind: faker.helpers.arrayElement([
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
      lapsed: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      releases: faker.number.int({ min: 0 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      text: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getListIndexerQueriesResponseMock = (
  overrideResponse: Partial<Extract<IndexersQueryList, object>> = {},
): IndexersQueryList => ({
  nextOffset: faker.helpers.arrayElement([faker.number.int(), null]),
  queries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      completedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      id: faker.string.uuid(),
      outcome: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["ok", "limited", "failing", "stopped", "lost"] as const),
        null,
      ]),
      releases: faker.number.int({ min: 0 }),
      reservedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      startedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      use: faker.helpers.arrayElement(["search", "recent", "fetch", "test"] as const),
    }),
  ),
  ...overrideResponse,
});

export const getTestIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersReport, object>> = {},
): IndexersReport => ({
  items: faker.number.int({ min: 0 }),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  sources: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getListTrackersResponseMock = (
  overrideResponse: Partial<Extract<IndexersTrackerList, object>> = {},
): IndexersTrackerList => ({
  trackers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
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
    }),
  ),
  ...overrideResponse,
});

export const getGetTrackerResponseMock = (
  overrideResponse: Partial<Extract<IndexersTracker, object>> = {},
): IndexersTracker => ({
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
  ...overrideResponse,
});

export const getRenameTrackerResponseMock = (
  overrideResponse: Partial<Extract<IndexersTracker, object>> = {},
): IndexersTracker => ({
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
  ...overrideResponse,
});

export const getMergeTrackerResponseMock = (
  overrideResponse: Partial<Extract<IndexersTracker, object>> = {},
): IndexersTracker => ({
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
  ...overrideResponse,
});

export const getGetIndexerCatalogMockHandler = (
  overrideResponse?:
    | IndexersIndexerCatalog
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexerCatalog> | IndexersIndexerCatalog),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexer-catalog",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetIndexerCatalogResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetCatalogAddressMockHandler = (
  overrideResponse?:
    | IndexersIndexerCatalog
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<IndexersIndexerCatalog> | IndexersIndexerCatalog),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/indexer-catalog/address",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetCatalogAddressResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddCustomDefinitionMockHandler = (
  overrideResponse?:
    | IndexersCatalogEntry
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersCatalogEntry> | IndexersCatalogEntry),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexer-catalog/custom",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddCustomDefinitionResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getDeleteCustomDefinitionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/indexer-catalog/custom/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetCatalogDefinitionMockHandler = (
  overrideResponse?:
    | IndexersDefinitionForm
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersDefinitionForm> | IndexersDefinitionForm),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexer-catalog/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetCatalogDefinitionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetIndexerSettingsMockHandler = (
  overrideResponse?:
    | IndexersSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersSettings> | IndexersSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexer-settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetIndexerSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateIndexerSettingsMockHandler = (
  overrideResponse?:
    | IndexersSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IndexersSettings> | IndexersSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/indexer-settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateIndexerSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListIndexersMockHandler = (
  overrideResponse?:
    | IndexersIndexerList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexerList> | IndexersIndexerList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListIndexersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateIndexerResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getTestIndexerDraftMockHandler = (
  overrideResponse?:
    | IndexersReport
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersReport> | IndexersReport),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestIndexerDraftResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteIndexerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/indexers/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetIndexerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/indexers/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateIndexerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListIndexerQueriesMockHandler = (
  overrideResponse?:
    | IndexersQueryList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersQueryList> | IndexersQueryList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers/:id/queries",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListIndexerQueriesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getTestIndexerMockHandler = (
  overrideResponse?:
    | IndexersReport
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersReport> | IndexersReport),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers/:id/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestIndexerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListTrackersMockHandler = (
  overrideResponse?:
    | IndexersTrackerList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersTrackerList> | IndexersTrackerList),
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

export const getGetTrackerMockHandler = (
  overrideResponse?:
    | IndexersTracker
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersTracker> | IndexersTracker),
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
    | IndexersTracker
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IndexersTracker> | IndexersTracker),
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

export const getMergeTrackerMockHandler = (
  overrideResponse?:
    | IndexersTracker
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersTracker> | IndexersTracker),
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
export const getIndexersMock = () => [
  getGetIndexerCatalogMockHandler(),
  getSetCatalogAddressMockHandler(),
  getAddCustomDefinitionMockHandler(),
  getDeleteCustomDefinitionMockHandler(),
  getGetCatalogDefinitionMockHandler(),
  getGetIndexerSettingsMockHandler(),
  getUpdateIndexerSettingsMockHandler(),
  getListIndexersMockHandler(),
  getCreateIndexerMockHandler(),
  getTestIndexerDraftMockHandler(),
  getDeleteIndexerMockHandler(),
  getGetIndexerMockHandler(),
  getUpdateIndexerMockHandler(),
  getListIndexerQueriesMockHandler(),
  getTestIndexerMockHandler(),
  getListTrackersMockHandler(),
  getGetTrackerMockHandler(),
  getRenameTrackerMockHandler(),
  getMergeTrackerMockHandler(),
];
