import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  ConnectionsOutputBody,
  EventsOutputBody,
  NotificationsBell,
  NotificationsConnection,
  NotificationsFeed,
  NotificationsTestPassed,
  ProvidersOutputBody,
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
  SettingsOutputBody,
} from "../../model";

export const getGetNotificationBellResponseMock = (
  overrideResponse: Partial<Extract<NotificationsBell, object>> = {},
): NotificationsBell => ({
  events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  warnings: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getUpdateNotificationBellResponseMock = (
  overrideResponse: Partial<Extract<NotificationsBell, object>> = {},
): NotificationsBell => ({
  events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  warnings: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getListNotificationConnectionsResponseMock = (
  overrideResponse: Partial<Extract<ConnectionsOutputBody, object>> = {},
): ConnectionsOutputBody => ({
  connections: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      node: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      on: faker.datatype.boolean(),
      posters: faker.datatype.boolean(),
      provider: faker.string.alpha({ length: { min: 10, max: 20 } }),
      sealed: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      shown: {
        [faker.string.alphanumeric(5)]: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      status: {
        givenUp: faker.number.int(),
        lastSent: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        reason: faker.string.alpha({ length: { min: 10, max: 20 } }),
        shown: faker.helpers.arrayElement([
          "off",
          "ok",
          "failing",
          "waiting",
          "paused",
          "holding",
        ] as const),
        since: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        until: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        waiting: faker.number.int(),
      },
      values: {
        [faker.string.alphanumeric(5)]: {},
      },
      warnings: faker.datatype.boolean(),
    }),
  ),
  ...overrideResponse,
});

export const getAddNotificationConnectionResponseMock = (
  overrideResponse: Partial<Extract<NotificationsConnection, object>> = {},
): NotificationsConnection => ({
  events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  node: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  on: faker.datatype.boolean(),
  posters: faker.datatype.boolean(),
  provider: faker.string.alpha({ length: { min: 10, max: 20 } }),
  sealed: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  shown: {
    [faker.string.alphanumeric(5)]: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  status: {
    givenUp: faker.number.int(),
    lastSent: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    reason: faker.string.alpha({ length: { min: 10, max: 20 } }),
    shown: faker.helpers.arrayElement([
      "off",
      "ok",
      "failing",
      "waiting",
      "paused",
      "holding",
    ] as const),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    waiting: faker.number.int(),
  },
  values: {
    [faker.string.alphanumeric(5)]: {},
  },
  warnings: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getTestNotificationDraftResponseMock = (
  overrideResponse: Partial<Extract<NotificationsTestPassed, object>> = {},
): NotificationsTestPassed => ({
  node: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getGetNotificationConnectionResponseMock = (
  overrideResponse: Partial<Extract<NotificationsConnection, object>> = {},
): NotificationsConnection => ({
  events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  node: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  on: faker.datatype.boolean(),
  posters: faker.datatype.boolean(),
  provider: faker.string.alpha({ length: { min: 10, max: 20 } }),
  sealed: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  shown: {
    [faker.string.alphanumeric(5)]: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  status: {
    givenUp: faker.number.int(),
    lastSent: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    reason: faker.string.alpha({ length: { min: 10, max: 20 } }),
    shown: faker.helpers.arrayElement([
      "off",
      "ok",
      "failing",
      "waiting",
      "paused",
      "holding",
    ] as const),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    waiting: faker.number.int(),
  },
  values: {
    [faker.string.alphanumeric(5)]: {},
  },
  warnings: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getUpdateNotificationConnectionResponseMock = (
  overrideResponse: Partial<Extract<NotificationsConnection, object>> = {},
): NotificationsConnection => ({
  events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  node: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  on: faker.datatype.boolean(),
  posters: faker.datatype.boolean(),
  provider: faker.string.alpha({ length: { min: 10, max: 20 } }),
  sealed: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  shown: {
    [faker.string.alphanumeric(5)]: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  status: {
    givenUp: faker.number.int(),
    lastSent: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    reason: faker.string.alpha({ length: { min: 10, max: 20 } }),
    shown: faker.helpers.arrayElement([
      "off",
      "ok",
      "failing",
      "waiting",
      "paused",
      "holding",
    ] as const),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    waiting: faker.number.int(),
  },
  values: {
    [faker.string.alphanumeric(5)]: {},
  },
  warnings: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getTestNotificationConnectionResponseMock = (
  overrideResponse: Partial<Extract<NotificationsTestPassed, object>> = {},
): NotificationsTestPassed => ({
  node: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListNotificationEventsResponseMock = (
  overrideResponse: Partial<Extract<EventsOutputBody, object>> = {},
): EventsOutputBody => ({
  events: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      bellByDefault: faker.datatype.boolean(),
      fixed: faker.datatype.boolean(),
      group: faker.helpers.arrayElement([
        "grabs",
        "imports",
        "library",
        "subtitles",
        "health",
        "backups",
        "moves",
        "system",
      ] as const),
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      onByDefault: faker.datatype.boolean(),
    }),
  ),
  ...overrideResponse,
});

export const getListNotificationsResponseMock = (
  overrideResponse: Partial<Extract<NotificationsFeed, object>> = {},
): NotificationsFeed => ({
  before: faker.helpers.arrayElement([faker.number.int(), null]),
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    at: faker.date.past().toISOString().slice(0, 19) + "Z",
    event: faker.string.alpha({ length: { min: 10, max: 20 } }),
    line: faker.string.alpha({ length: { min: 10, max: 20 } }),
    link: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    seq: faker.number.int(),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    unread: faker.helpers.arrayElement([faker.datatype.boolean(), null]),
  })),
  unread: faker.helpers.arrayElement([faker.number.int(), null]),
  ...overrideResponse,
});

export const getListNotificationProvidersResponseMock = (
  overrideResponse: Partial<Extract<ProvidersOutputBody, object>> = {},
): ProvidersOutputBody => ({
  providers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean() ? { default: {} } : {}),
          details: faker.datatype.boolean(),
          help: faker.string.alpha({ length: { min: 10, max: 20 } }),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          kind: faker.helpers.arrayElement([
            "text",
            "url",
            "number",
            "select",
            "switch",
            "list",
            "headers",
            "node",
            "duration",
          ] as const),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          ...(faker.datatype.boolean() ? { max: faker.number.int() } : {}),
          ...(faker.datatype.boolean() ? { min: faker.number.int() } : {}),
          optional: faker.datatype.boolean(),
          options: faker.helpers.arrayElement([
            Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
              () => ({
                label: faker.string.alpha({ length: { min: 10, max: 20 } }),
                value: faker.string.alpha({ length: { min: 10, max: 20 } }),
              }),
            ),
            null,
          ]),
          secret: faker.datatype.boolean(),
        }),
      ),
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      image: faker.helpers.arrayElement(["none", "bytes", "publicUrl", "bodyUrl"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getGetNotificationSettingsResponseSettingDurationFieldMock = (
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

export const getGetNotificationSettingsResponseSettingCountFieldMock = (
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

export const getGetNotificationSettingsResponseSettingChoiceFieldMock = (
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

export const getGetNotificationSettingsResponseSettingWindowFieldMock = (
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

export const getGetNotificationSettingsResponseSettingListenFieldMock = (
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

export const getGetNotificationSettingsResponseSettingSwitchFieldMock = (
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

export const getGetNotificationSettingsResponseSettingRateFieldMock = (
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

export const getGetNotificationSettingsResponseSettingAddressFieldMock = (
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

export const getGetNotificationSettingsResponseSettingFolderFieldMock = (
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

export const getGetNotificationSettingsResponseSettingSpaceFieldMock = (
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

export const getGetNotificationSettingsResponseSettingTemplateFieldMock = (
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

export const getGetNotificationSettingsResponseSettingListFieldMock = (
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

export const getGetNotificationSettingsResponseSettingSizeFieldMock = (
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

export const getGetNotificationSettingsResponseSettingIPFieldMock = (
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

export const getGetNotificationSettingsResponseMock = (
  overrideResponse: Partial<Extract<SettingsOutputBody, object>> = {},
): SettingsOutputBody => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetNotificationSettingsResponseSettingDurationFieldMock() },
      { ...getGetNotificationSettingsResponseSettingCountFieldMock() },
      { ...getGetNotificationSettingsResponseSettingChoiceFieldMock() },
      { ...getGetNotificationSettingsResponseSettingWindowFieldMock() },
      { ...getGetNotificationSettingsResponseSettingListenFieldMock() },
      { ...getGetNotificationSettingsResponseSettingSwitchFieldMock() },
      { ...getGetNotificationSettingsResponseSettingRateFieldMock() },
      { ...getGetNotificationSettingsResponseSettingAddressFieldMock() },
      { ...getGetNotificationSettingsResponseSettingFolderFieldMock() },
      { ...getGetNotificationSettingsResponseSettingSpaceFieldMock() },
      { ...getGetNotificationSettingsResponseSettingTemplateFieldMock() },
      { ...getGetNotificationSettingsResponseSettingListFieldMock() },
      { ...getGetNotificationSettingsResponseSettingSizeFieldMock() },
      { ...getGetNotificationSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateNotificationSettingsResponseSettingDurationFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingCountFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingChoiceFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingWindowFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingListenFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingSwitchFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingRateFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingAddressFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingFolderFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingSpaceFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingTemplateFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingListFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingSizeFieldMock = (
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

export const getUpdateNotificationSettingsResponseSettingIPFieldMock = (
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

export const getUpdateNotificationSettingsResponseMock = (
  overrideResponse: Partial<Extract<SettingsOutputBody, object>> = {},
): SettingsOutputBody => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateNotificationSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingCountFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingListenFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingRateFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingSpaceFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingTemplateFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingListFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingSizeFieldMock() },
      { ...getUpdateNotificationSettingsResponseSettingIPFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getGetNotificationBellMockHandler = (
  overrideResponse?:
    | NotificationsBell
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NotificationsBell> | NotificationsBell),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/bell",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNotificationBellResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateNotificationBellMockHandler = (
  overrideResponse?:
    | NotificationsBell
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<NotificationsBell> | NotificationsBell),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/notifications/bell",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateNotificationBellResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListNotificationConnectionsMockHandler = (
  overrideResponse?:
    | ConnectionsOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ConnectionsOutputBody> | ConnectionsOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/connections",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListNotificationConnectionsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddNotificationConnectionMockHandler = (
  overrideResponse?:
    | NotificationsConnection
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NotificationsConnection> | NotificationsConnection),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/notifications/connections",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddNotificationConnectionResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getTestNotificationDraftMockHandler = (
  overrideResponse?:
    | NotificationsTestPassed
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NotificationsTestPassed> | NotificationsTestPassed),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/notifications/connections/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestNotificationDraftResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteNotificationConnectionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/notifications/connections/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetNotificationConnectionMockHandler = (
  overrideResponse?:
    | NotificationsConnection
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NotificationsConnection> | NotificationsConnection),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/connections/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNotificationConnectionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateNotificationConnectionMockHandler = (
  overrideResponse?:
    | NotificationsConnection
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<NotificationsConnection> | NotificationsConnection),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/notifications/connections/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateNotificationConnectionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getTestNotificationConnectionMockHandler = (
  overrideResponse?:
    | NotificationsTestPassed
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NotificationsTestPassed> | NotificationsTestPassed),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/notifications/connections/:id/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestNotificationConnectionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListNotificationEventsMockHandler = (
  overrideResponse?:
    | EventsOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<EventsOutputBody> | EventsOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/events",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListNotificationEventsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListNotificationsMockHandler = (
  overrideResponse?:
    | NotificationsFeed
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NotificationsFeed> | NotificationsFeed),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/feed",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListNotificationsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getMarkNotificationsSeenMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/notifications/feed/seen",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListNotificationProvidersMockHandler = (
  overrideResponse?:
    | ProvidersOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ProvidersOutputBody> | ProvidersOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/providers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListNotificationProvidersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetNotificationSettingsMockHandler = (
  overrideResponse?:
    | SettingsOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<SettingsOutputBody> | SettingsOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/notifications/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNotificationSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateNotificationSettingsMockHandler = (
  overrideResponse?:
    | SettingsOutputBody
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<SettingsOutputBody> | SettingsOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/notifications/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateNotificationSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getNotificationsMock = () => [
  getGetNotificationBellMockHandler(),
  getUpdateNotificationBellMockHandler(),
  getListNotificationConnectionsMockHandler(),
  getAddNotificationConnectionMockHandler(),
  getTestNotificationDraftMockHandler(),
  getDeleteNotificationConnectionMockHandler(),
  getGetNotificationConnectionMockHandler(),
  getUpdateNotificationConnectionMockHandler(),
  getTestNotificationConnectionMockHandler(),
  getListNotificationEventsMockHandler(),
  getListNotificationsMockHandler(),
  getMarkNotificationsSeenMockHandler(),
  getListNotificationProvidersMockHandler(),
  getGetNotificationSettingsMockHandler(),
  getUpdateNotificationSettingsMockHandler(),
];
