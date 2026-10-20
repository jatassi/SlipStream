import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingPortField,
  SettingSwitchField,
  SettingWindowField,
  WorksAdded,
  WorksCandidateList,
  WorksContentType,
  WorksContentTypeList,
  WorksDateValue,
  WorksDurationValue,
  WorksLinkValue,
  WorksNumberValue,
  WorksPeopleValue,
  WorksTextValue,
  WorksTextsValue,
  WorksTree,
  WorksTypeSettings,
  WorksWorkList,
} from "../../model";

export const getListContentTypesResponseMock = (
  overrideResponse: Partial<Extract<WorksContentTypeList, object>> = {},
): WorksContentTypeList => ({
  contentTypes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      enabled: faker.datatype.boolean(),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          kind: faker.helpers.arrayElement([
            "text",
            "number",
            "date",
            "duration",
            "texts",
            "people",
            "link",
          ] as const),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
        }),
      ),
      icon: faker.helpers.arrayElement([
        "film",
        "tv",
        "note",
        "open-book",
        "headphones",
        "panels",
        "image",
        "shapes",
      ] as const),
      id: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      item: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          many: faker.string.alpha({ length: { min: 10, max: 20 } }),
          one: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      work: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
    }),
  ),
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseMock = (
  overrideResponse: Partial<Extract<WorksCandidateList, object>> = {},
): WorksCandidateList => ({
  candidates: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getSearchCatalogResponseWorksTextValueMock() },
            { ...getSearchCatalogResponseWorksNumberValueMock() },
            { ...getSearchCatalogResponseWorksDateValueMock() },
            { ...getSearchCatalogResponseWorksDurationValueMock() },
            { ...getSearchCatalogResponseWorksTextsValueMock() },
            { ...getSearchCatalogResponseWorksPeopleValueMock() },
            { ...getSearchCatalogResponseWorksLinkValueMock() },
          ]),
      ),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    }),
  ),
  ...overrideResponse,
});

export const getSetContentTypeEnabledResponseMock = (
  overrideResponse: Partial<Extract<WorksContentType, object>> = {},
): WorksContentType => ({
  enabled: faker.datatype.boolean(),
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement([
        "text",
        "number",
        "date",
        "duration",
        "texts",
        "people",
        "link",
      ] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
    }),
  ),
  icon: faker.helpers.arrayElement([
    "film",
    "tv",
    "note",
    "open-book",
    "headphones",
    "panels",
    "image",
    "shapes",
  ] as const),
  id: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  item: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  work: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetContentTypeSettingsResponseSettingDurationFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingCountFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingChoiceFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingWindowFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingPortFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingSwitchFieldMock = (
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

export const getGetContentTypeSettingsResponseMock = (
  overrideResponse: Partial<Extract<WorksTypeSettings, object>> = {},
): WorksTypeSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetContentTypeSettingsResponseSettingDurationFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingCountFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingChoiceFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingWindowFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingPortFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingSwitchFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateContentTypeSettingsResponseSettingDurationFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingCountFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingChoiceFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingWindowFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingPortFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingSwitchFieldMock = (
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

export const getUpdateContentTypeSettingsResponseMock = (
  overrideResponse: Partial<Extract<WorksTypeSettings, object>> = {},
): WorksTypeSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateContentTypeSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingCountFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingPortFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingSwitchFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getListWorksResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListWorksResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getListWorksResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getListWorksResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getListWorksResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getListWorksResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getListWorksResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getListWorksResponseMock = (
  overrideResponse: Partial<Extract<WorksWorkList, object>> = {},
): WorksWorkList => ({
  works: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getListWorksResponseWorksTextValueMock() },
        { ...getListWorksResponseWorksNumberValueMock() },
        { ...getListWorksResponseWorksDateValueMock() },
        { ...getListWorksResponseWorksDurationValueMock() },
        { ...getListWorksResponseWorksTextsValueMock() },
        { ...getListWorksResponseWorksPeopleValueMock() },
        { ...getListWorksResponseWorksLinkValueMock() },
      ]),
    ),
    goneUpstreamAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  ...overrideResponse,
});

export const getAddWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getAddWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksAdded, object>> = {},
): WorksAdded => ({
  created: faker.datatype.boolean(),
  work: {
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getAddWorkResponseWorksTextValueMock() },
        { ...getAddWorkResponseWorksNumberValueMock() },
        { ...getAddWorkResponseWorksDateValueMock() },
        { ...getAddWorkResponseWorksDurationValueMock() },
        { ...getAddWorkResponseWorksTextsValueMock() },
        { ...getAddWorkResponseWorksPeopleValueMock() },
        { ...getAddWorkResponseWorksLinkValueMock() },
      ]),
    ),
    goneUpstreamAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    id: faker.string.uuid(),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        availability: {
          basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
          date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
        },
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getAddWorkResponseWorksTextValueMock() },
              { ...getAddWorkResponseWorksNumberValueMock() },
              { ...getAddWorkResponseWorksDateValueMock() },
              { ...getAddWorkResponseWorksDurationValueMock() },
              { ...getAddWorkResponseWorksTextsValueMock() },
              { ...getAddWorkResponseWorksPeopleValueMock() },
              { ...getAddWorkResponseWorksLinkValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        monitored: faker.datatype.boolean(),
        position: faker.number.int({ min: 1 }),
        ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            scale: faker.number.float({ min: 0, fractionDigits: 2 }),
            source: faker.string.alpha({ length: { min: 10, max: 20 } }),
            value: faker.number.float({ min: 0, fractionDigits: 2 }),
            votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
          }),
        ),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        depth: faker.number.int({ min: 1 }),
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getAddWorkResponseWorksTextValueMock() },
              { ...getAddWorkResponseWorksNumberValueMock() },
              { ...getAddWorkResponseWorksDateValueMock() },
              { ...getAddWorkResponseWorksDurationValueMock() },
              { ...getAddWorkResponseWorksTextsValueMock() },
              { ...getAddWorkResponseWorksPeopleValueMock() },
              { ...getAddWorkResponseWorksLinkValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        position: faker.number.int({ min: 1 }),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
        votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      }),
    ),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getGetWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksTree, object>> = {},
): WorksTree => ({
  addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkResponseWorksTextValueMock() },
      { ...getGetWorkResponseWorksNumberValueMock() },
      { ...getGetWorkResponseWorksDateValueMock() },
      { ...getGetWorkResponseWorksDurationValueMock() },
      { ...getGetWorkResponseWorksTextsValueMock() },
      { ...getGetWorkResponseWorksPeopleValueMock() },
      { ...getGetWorkResponseWorksLinkValueMock() },
    ]),
  ),
  goneUpstreamAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    availability: {
      basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
      date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    },
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getGetWorkResponseWorksTextValueMock() },
        { ...getGetWorkResponseWorksNumberValueMock() },
        { ...getGetWorkResponseWorksDateValueMock() },
        { ...getGetWorkResponseWorksDurationValueMock() },
        { ...getGetWorkResponseWorksTextsValueMock() },
        { ...getGetWorkResponseWorksPeopleValueMock() },
        { ...getGetWorkResponseWorksLinkValueMock() },
      ]),
    ),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    monitored: faker.datatype.boolean(),
    position: faker.number.int({ min: 1 }),
    ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
        votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      }),
    ),
    withdrawnAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
  })),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      depth: faker.number.int({ min: 1 }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getGetWorkResponseWorksTextValueMock() },
            { ...getGetWorkResponseWorksNumberValueMock() },
            { ...getGetWorkResponseWorksDateValueMock() },
            { ...getGetWorkResponseWorksDurationValueMock() },
            { ...getGetWorkResponseWorksTextsValueMock() },
            { ...getGetWorkResponseWorksPeopleValueMock() },
            { ...getGetWorkResponseWorksLinkValueMock() },
          ]),
      ),
      id: faker.string.uuid(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      position: faker.number.int({ min: 1 }),
      withdrawnAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
    }),
  ),
  ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      scale: faker.number.float({ min: 0, fractionDigits: 2 }),
      source: faker.string.alpha({ length: { min: 10, max: 20 } }),
      value: faker.number.float({ min: 0, fractionDigits: 2 }),
      votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
    }),
  ),
  refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksTree, object>> = {},
): WorksTree => ({
  addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getRefreshWorkResponseWorksTextValueMock() },
      { ...getRefreshWorkResponseWorksNumberValueMock() },
      { ...getRefreshWorkResponseWorksDateValueMock() },
      { ...getRefreshWorkResponseWorksDurationValueMock() },
      { ...getRefreshWorkResponseWorksTextsValueMock() },
      { ...getRefreshWorkResponseWorksPeopleValueMock() },
      { ...getRefreshWorkResponseWorksLinkValueMock() },
    ]),
  ),
  goneUpstreamAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    availability: {
      basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
      date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    },
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getRefreshWorkResponseWorksTextValueMock() },
        { ...getRefreshWorkResponseWorksNumberValueMock() },
        { ...getRefreshWorkResponseWorksDateValueMock() },
        { ...getRefreshWorkResponseWorksDurationValueMock() },
        { ...getRefreshWorkResponseWorksTextsValueMock() },
        { ...getRefreshWorkResponseWorksPeopleValueMock() },
        { ...getRefreshWorkResponseWorksLinkValueMock() },
      ]),
    ),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    monitored: faker.datatype.boolean(),
    position: faker.number.int({ min: 1 }),
    ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
        votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      }),
    ),
    withdrawnAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
  })),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      depth: faker.number.int({ min: 1 }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getRefreshWorkResponseWorksTextValueMock() },
            { ...getRefreshWorkResponseWorksNumberValueMock() },
            { ...getRefreshWorkResponseWorksDateValueMock() },
            { ...getRefreshWorkResponseWorksDurationValueMock() },
            { ...getRefreshWorkResponseWorksTextsValueMock() },
            { ...getRefreshWorkResponseWorksPeopleValueMock() },
            { ...getRefreshWorkResponseWorksLinkValueMock() },
          ]),
      ),
      id: faker.string.uuid(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      position: faker.number.int({ min: 1 }),
      withdrawnAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
    }),
  ),
  ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      scale: faker.number.float({ min: 0, fractionDigits: 2 }),
      source: faker.string.alpha({ length: { min: 10, max: 20 } }),
      value: faker.number.float({ min: 0, fractionDigits: 2 }),
      votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
    }),
  ),
  refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListContentTypesMockHandler = (
  overrideResponse?:
    | WorksContentTypeList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksContentTypeList> | WorksContentTypeList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListContentTypesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSearchCatalogMockHandler = (
  overrideResponse?:
    | WorksCandidateList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksCandidateList> | WorksCandidateList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types/:contentType/catalog",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSearchCatalogResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetContentTypeEnabledMockHandler = (
  overrideResponse?:
    | WorksContentType
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksContentType> | WorksContentType),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/content-types/:contentType/enabled",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetContentTypeEnabledResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetContentTypeSettingsMockHandler = (
  overrideResponse?:
    | WorksTypeSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksTypeSettings> | WorksTypeSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types/:contentType/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetContentTypeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateContentTypeSettingsMockHandler = (
  overrideResponse?:
    | WorksTypeSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<WorksTypeSettings> | WorksTypeSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/content-types/:contentType/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateContentTypeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListWorksMockHandler = (
  overrideResponse?:
    | WorksWorkList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksWorkList> | WorksWorkList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListWorksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddWorkMockHandler = (
  overrideResponse?:
    | WorksAdded
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<WorksAdded> | WorksAdded),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetWorkMockHandler = (
  overrideResponse?:
    | WorksTree
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<WorksTree> | WorksTree),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRefreshWorkMockHandler = (
  overrideResponse?:
    | WorksTree
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<WorksTree> | WorksTree),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/:workId/refresh",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRefreshWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getWorksMock = () => [
  getListContentTypesMockHandler(),
  getSearchCatalogMockHandler(),
  getSetContentTypeEnabledMockHandler(),
  getGetContentTypeSettingsMockHandler(),
  getUpdateContentTypeSettingsMockHandler(),
  getListWorksMockHandler(),
  getAddWorkMockHandler(),
  getGetWorkMockHandler(),
  getRefreshWorkMockHandler(),
];
