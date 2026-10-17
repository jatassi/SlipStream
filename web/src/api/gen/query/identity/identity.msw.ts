import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  IdentityAccountDetail,
  IdentityAccountRow,
  IdentityAccountRows,
  IdentityAddress,
  IdentityApiKeyMade,
  IdentityCeremony,
  IdentityClaimed,
  IdentityDeadLink,
  IdentityEmailChangeStarted,
  IdentityIssuedResetLink,
  IdentityIssuedSignupLink,
  IdentityLinkView,
  IdentityLiveLink,
  IdentityPasskey,
  IdentityPasskeyRows,
  IdentityPendingLinks,
  IdentityRegeneratedAPIKey,
  IdentitySession,
  IdentitySessionRows,
  IdentitySettings,
  IdentitySignedIn,
  IdentitySignupLinkURL,
} from "../../model";

export const getUpdateOwnAccountResponseMock = (
  overrideResponse: Partial<Extract<IdentitySession, object>> = {},
): IdentitySession => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  keyPrompt: faker.helpers.arrayElement(["ask", "never"] as const),
  pinRequired: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getStartEmailChangeResponseMock = (
  overrideResponse: Partial<Extract<IdentityEmailChangeStarted, object>> = {},
): IdentityEmailChangeStarted => ({
  email: {
    ...(faker.datatype.boolean()
      ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    outcome: faker.helpers.arrayElement(["sent", "failed", "not-set-up"] as const),
  },
  ...overrideResponse,
});

export const getListAccountsResponseMock = (
  overrideResponse: Partial<Extract<IdentityAccountRows, object>> = {},
): IdentityAccountRows => ({
  accounts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      admin: faker.datatype.boolean(),
      disabled: faker.datatype.boolean(),
      displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
      email: faker.internet.email(),
      id: faker.string.uuid(),
      ...(faker.datatype.boolean()
        ? { lastSignInAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      passkeyCount: faker.number.int(),
      pin: {
        state: faker.helpers.arrayElement(["fine", "waiting", "off"] as const),
        ...(faker.datatype.boolean()
          ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
          : {}),
      },
    }),
  ),
  ...overrideResponse,
});

export const getGetAccountResponseMock = (
  overrideResponse: Partial<Extract<IdentityAccountDetail, object>> = {},
): IdentityAccountDetail => ({
  account: {
    admin: faker.datatype.boolean(),
    disabled: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
    ...(faker.datatype.boolean()
      ? { lastSignInAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    passkeyCount: faker.number.int(),
    pin: {
      state: faker.helpers.arrayElement(["fine", "waiting", "off"] as const),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
    },
  },
  passkeys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      host: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.uuid(),
      ...(faker.datatype.boolean()
        ? { lastUsedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  sessions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      current: faker.datatype.boolean(),
      device: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.uuid(),
      lastSeenAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  ...overrideResponse,
});

export const getUpdateAccountResponseMock = (
  overrideResponse: Partial<Extract<IdentityAccountRow, object>> = {},
): IdentityAccountRow => ({
  admin: faker.datatype.boolean(),
  disabled: faker.datatype.boolean(),
  displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
  email: faker.internet.email(),
  id: faker.string.uuid(),
  ...(faker.datatype.boolean()
    ? { lastSignInAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  passkeyCount: faker.number.int(),
  pin: {
    state: faker.helpers.arrayElement(["fine", "waiting", "off"] as const),
    ...(faker.datatype.boolean()
      ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
  },
  ...overrideResponse,
});

export const getCreateResetLinkResponseMock = (
  overrideResponse: Partial<Extract<IdentityIssuedResetLink, object>> = {},
): IdentityIssuedResetLink => ({
  email: {
    ...(faker.datatype.boolean()
      ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    outcome: faker.helpers.arrayElement(["sent", "failed", "not-set-up"] as const),
  },
  url: faker.internet.url(),
  ...overrideResponse,
});

export const getListAccountSessionsResponseMock = (
  overrideResponse: Partial<Extract<IdentitySessionRows, object>> = {},
): IdentitySessionRows => ({
  sessions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      current: faker.datatype.boolean(),
      device: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.uuid(),
      lastSeenAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  ...overrideResponse,
});

export const getGetAPIKeyResponseMock = (
  overrideResponse: Partial<Extract<IdentityApiKeyMade, object>> = {},
): IdentityApiKeyMade => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getRegenerateAPIKeyResponseMock = (
  overrideResponse: Partial<Extract<IdentityRegeneratedAPIKey, object>> = {},
): IdentityRegeneratedAPIKey => ({
  apiKey: faker.helpers.fromRegExp("^ssk_[A-Za-z0-9_-]{43}$"),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getGetEmailLinkByTokenResponseIdentityLiveLinkMock = (
  overrideResponse: Partial<IdentityLiveLink> = {},
): IdentityLiveLink => ({
  ...{
    ...(faker.datatype.boolean()
      ? { adminName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    serverName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    state: faker.helpers.arrayElement(["live"] as const),
  },
  ...overrideResponse,
});

export const getGetEmailLinkByTokenResponseIdentityDeadLinkMock = (
  overrideResponse: Partial<IdentityDeadLink> = {},
): IdentityDeadLink => ({
  ...{
    ...(faker.datatype.boolean()
      ? { adminName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    ...(faker.datatype.boolean()
      ? { issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    reason: faker.helpers.arrayElement(["unknown", "withdrawn", "expired", "used"] as const),
    state: faker.helpers.arrayElement(["dead"] as const),
  },
  ...overrideResponse,
});

export const getGetEmailLinkByTokenResponseMock = (): IdentityLinkView =>
  faker.helpers.arrayElement([
    { ...getGetEmailLinkByTokenResponseIdentityLiveLinkMock() },
    { ...getGetEmailLinkByTokenResponseIdentityDeadLinkMock() },
  ]);

export const getGetAddressResponseMock = (
  overrideResponse: Partial<Extract<IdentityAddress, object>> = {},
): IdentityAddress => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  suggestedAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getSetAddressResponseMock = (
  overrideResponse: Partial<Extract<IdentityAddress, object>> = {},
): IdentityAddress => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  suggestedAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getGetIdentitySettingsResponseMock = (
  overrideResponse: Partial<Extract<IdentitySettings, object>> = {},
): IdentitySettings => ({
  addressFailures: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
  },
  addressWindow: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  emailLinkLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  pinOffAfter: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
  },
  pinWait: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  pinWaitAfter: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
  },
  reconfirmWindow: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  resetLinkLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  sessionLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  signupLinkLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateIdentitySettingsResponseMock = (
  overrideResponse: Partial<Extract<IdentitySettings, object>> = {},
): IdentitySettings => ({
  addressFailures: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
  },
  addressWindow: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  emailLinkLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  pinOffAfter: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
  },
  pinWait: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  pinWaitAfter: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
  },
  reconfirmWindow: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  resetLinkLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  sessionLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  signupLinkLifetime: {
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
    max: faker.number.int(),
    min: faker.number.int(),
    source: faker.helpers.arrayElement(["database"] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getListPasskeysResponseMock = (
  overrideResponse: Partial<Extract<IdentityPasskeyRows, object>> = {},
): IdentityPasskeyRows => ({
  passkeys: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      host: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.uuid(),
      ...(faker.datatype.boolean()
        ? { lastUsedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getStartPasskeySignInResponseMock = (
  overrideResponse: Partial<Extract<IdentityCeremony, object>> = {},
): IdentityCeremony => ({
  ceremony: faker.string.alpha({ length: { min: 10, max: 20 } }),
  options: {},
  ...overrideResponse,
});

export const getFinishPasskeySignInResponseMock = (
  overrideResponse: Partial<Extract<IdentitySignedIn, object>> = {},
): IdentitySignedIn => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  keyPrompt: faker.helpers.arrayElement(["ask", "never"] as const),
  passkeyPrompt: faker.datatype.boolean(),
  pinRequired: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getStartPasskeyRegistrationResponseMock = (
  overrideResponse: Partial<Extract<IdentityCeremony, object>> = {},
): IdentityCeremony => ({
  ceremony: faker.string.alpha({ length: { min: 10, max: 20 } }),
  options: {},
  ...overrideResponse,
});

export const getFinishPasskeyRegistrationResponseMock = (
  overrideResponse: Partial<Extract<IdentityPasskey, object>> = {},
): IdentityPasskey => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  host: faker.string.alpha({ length: { min: 10, max: 20 } }),
  id: faker.string.uuid(),
  ...(faker.datatype.boolean()
    ? { lastUsedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getUpdatePasskeyResponseMock = (
  overrideResponse: Partial<Extract<IdentityPasskey, object>> = {},
): IdentityPasskey => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  host: faker.string.alpha({ length: { min: 10, max: 20 } }),
  id: faker.string.uuid(),
  ...(faker.datatype.boolean()
    ? { lastUsedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getGetResetLinkByTokenResponseIdentityLiveLinkMock = (
  overrideResponse: Partial<IdentityLiveLink> = {},
): IdentityLiveLink => ({
  ...{
    ...(faker.datatype.boolean()
      ? { adminName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    serverName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    state: faker.helpers.arrayElement(["live"] as const),
  },
  ...overrideResponse,
});

export const getGetResetLinkByTokenResponseIdentityDeadLinkMock = (
  overrideResponse: Partial<IdentityDeadLink> = {},
): IdentityDeadLink => ({
  ...{
    ...(faker.datatype.boolean()
      ? { adminName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    ...(faker.datatype.boolean()
      ? { issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    reason: faker.helpers.arrayElement(["unknown", "withdrawn", "expired", "used"] as const),
    state: faker.helpers.arrayElement(["dead"] as const),
  },
  ...overrideResponse,
});

export const getGetResetLinkByTokenResponseMock = (): IdentityLinkView =>
  faker.helpers.arrayElement([
    { ...getGetResetLinkByTokenResponseIdentityLiveLinkMock() },
    { ...getGetResetLinkByTokenResponseIdentityDeadLinkMock() },
  ]);

export const getRedeemResetLinkResponseMock = (
  overrideResponse: Partial<Extract<IdentitySignedIn, object>> = {},
): IdentitySignedIn => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  keyPrompt: faker.helpers.arrayElement(["ask", "never"] as const),
  passkeyPrompt: faker.datatype.boolean(),
  pinRequired: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getGetSessionResponseMock = (
  overrideResponse: Partial<Extract<IdentitySession, object>> = {},
): IdentitySession => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  keyPrompt: faker.helpers.arrayElement(["ask", "never"] as const),
  pinRequired: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getCreateSessionResponseMock = (
  overrideResponse: Partial<Extract<IdentitySignedIn, object>> = {},
): IdentitySignedIn => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  keyPrompt: faker.helpers.arrayElement(["ask", "never"] as const),
  passkeyPrompt: faker.datatype.boolean(),
  pinRequired: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getStartPasskeyReconfirmResponseMock = (
  overrideResponse: Partial<Extract<IdentityCeremony, object>> = {},
): IdentityCeremony => ({
  ceremony: faker.string.alpha({ length: { min: 10, max: 20 } }),
  options: {},
  ...overrideResponse,
});

export const getListSessionsResponseMock = (
  overrideResponse: Partial<Extract<IdentitySessionRows, object>> = {},
): IdentitySessionRows => ({
  sessions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      current: faker.datatype.boolean(),
      device: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.uuid(),
      lastSeenAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  ...overrideResponse,
});

export const getClaimControllerResponseMock = (
  overrideResponse: Partial<Extract<IdentityClaimed, object>> = {},
): IdentityClaimed => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  apiKey: faker.helpers.fromRegExp("^ssk_[A-Za-z0-9_-]{43}$"),
  ...overrideResponse,
});

export const getListSignupLinksResponseMock = (
  overrideResponse: Partial<Extract<IdentityPendingLinks, object>> = {},
): IdentityPendingLinks => ({
  links: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    id: faker.string.uuid(),
    issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    ...(faker.datatype.boolean()
      ? { issuedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  })),
  ...overrideResponse,
});

export const getCreateSignupLinkResponseMock = (
  overrideResponse: Partial<Extract<IdentityIssuedSignupLink, object>> = {},
): IdentityIssuedSignupLink => ({
  email: {
    ...(faker.datatype.boolean()
      ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    outcome: faker.helpers.arrayElement(["sent", "failed", "not-set-up"] as const),
  },
  link: {
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    id: faker.string.uuid(),
    issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    ...(faker.datatype.boolean()
      ? { issuedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  url: faker.internet.url(),
  ...overrideResponse,
});

export const getGetSignupLinkByTokenResponseIdentityLiveLinkMock = (
  overrideResponse: Partial<IdentityLiveLink> = {},
): IdentityLiveLink => ({
  ...{
    ...(faker.datatype.boolean()
      ? { adminName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    serverName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    state: faker.helpers.arrayElement(["live"] as const),
  },
  ...overrideResponse,
});

export const getGetSignupLinkByTokenResponseIdentityDeadLinkMock = (
  overrideResponse: Partial<IdentityDeadLink> = {},
): IdentityDeadLink => ({
  ...{
    ...(faker.datatype.boolean()
      ? { adminName: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    ...(faker.datatype.boolean()
      ? { issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    reason: faker.helpers.arrayElement(["unknown", "withdrawn", "expired", "used"] as const),
    state: faker.helpers.arrayElement(["dead"] as const),
  },
  ...overrideResponse,
});

export const getGetSignupLinkByTokenResponseMock = (): IdentityLinkView =>
  faker.helpers.arrayElement([
    { ...getGetSignupLinkByTokenResponseIdentityLiveLinkMock() },
    { ...getGetSignupLinkByTokenResponseIdentityDeadLinkMock() },
  ]);

export const getRedeemSignupLinkResponseMock = (
  overrideResponse: Partial<Extract<IdentitySignedIn, object>> = {},
): IdentitySignedIn => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
  keyPrompt: faker.helpers.arrayElement(["ask", "never"] as const),
  passkeyPrompt: faker.datatype.boolean(),
  pinRequired: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getGetSignupLinkResponseMock = (
  overrideResponse: Partial<Extract<IdentitySignupLinkURL, object>> = {},
): IdentitySignupLinkURL => ({
  link: {
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    id: faker.string.uuid(),
    issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    ...(faker.datatype.boolean()
      ? { issuedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  url: faker.internet.url(),
  ...overrideResponse,
});

export const getResendSignupLinkResponseMock = (
  overrideResponse: Partial<Extract<IdentityIssuedSignupLink, object>> = {},
): IdentityIssuedSignupLink => ({
  email: {
    ...(faker.datatype.boolean()
      ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    outcome: faker.helpers.arrayElement(["sent", "failed", "not-set-up"] as const),
  },
  link: {
    email: faker.internet.email(),
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    id: faker.string.uuid(),
    issuedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    ...(faker.datatype.boolean()
      ? { issuedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  url: faker.internet.url(),
  ...overrideResponse,
});

export const getUpdateOwnAccountMockHandler = (
  overrideResponse?:
    | IdentitySession
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IdentitySession> | IdentitySession),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/account",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateOwnAccountResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartEmailChangeMockHandler = (
  overrideResponse?:
    | IdentityEmailChangeStarted
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityEmailChangeStarted> | IdentityEmailChangeStarted),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/account/email",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartEmailChangeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getChangePINMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/account/pin",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListAccountsMockHandler = (
  overrideResponse?:
    | IdentityAccountRows
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityAccountRows> | IdentityAccountRows),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/accounts",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListAccountsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteAccountMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/accounts/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetAccountMockHandler = (
  overrideResponse?:
    | IdentityAccountDetail
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityAccountDetail> | IdentityAccountDetail),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/accounts/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetAccountResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateAccountMockHandler = (
  overrideResponse?:
    | IdentityAccountRow
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IdentityAccountRow> | IdentityAccountRow),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/accounts/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateAccountResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateResetLinkMockHandler = (
  overrideResponse?:
    | IdentityIssuedResetLink
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityIssuedResetLink> | IdentityIssuedResetLink),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/accounts/:id/reset-link",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateResetLinkResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getEndAccountSessionsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/accounts/:id/sessions",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListAccountSessionsMockHandler = (
  overrideResponse?:
    | IdentitySessionRows
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentitySessionRows> | IdentitySessionRows),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/accounts/:id/sessions",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListAccountSessionsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getEndAccountSessionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/accounts/:id/sessions/:session",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetAPIKeyMockHandler = (
  overrideResponse?:
    | IdentityApiKeyMade
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityApiKeyMade> | IdentityApiKeyMade),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/api-key",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetAPIKeyResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRegenerateAPIKeyMockHandler = (
  overrideResponse?:
    | IdentityRegeneratedAPIKey
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityRegeneratedAPIKey> | IdentityRegeneratedAPIKey),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/api-key",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRegenerateAPIKeyResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetEmailLinkByTokenMockHandler = (
  overrideResponse?:
    | IdentityLinkView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityLinkView> | IdentityLinkView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/email-links/by-token/:token",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetEmailLinkByTokenResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getConfirmEmailMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/email-links/by-token/:token/confirm",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetAddressMockHandler = (
  overrideResponse?:
    | IdentityAddress
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityAddress> | IdentityAddress),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/identity/address",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetAddressResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetAddressMockHandler = (
  overrideResponse?:
    | IdentityAddress
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<IdentityAddress> | IdentityAddress),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/identity/address",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetAddressResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetIdentitySettingsMockHandler = (
  overrideResponse?:
    | IdentitySettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentitySettings> | IdentitySettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/identity/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetIdentitySettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateIdentitySettingsMockHandler = (
  overrideResponse?:
    | IdentitySettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IdentitySettings> | IdentitySettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/identity/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateIdentitySettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListPasskeysMockHandler = (
  overrideResponse?:
    | IdentityPasskeyRows
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityPasskeyRows> | IdentityPasskeyRows),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/passkeys",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListPasskeysResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartPasskeySignInMockHandler = (
  overrideResponse?:
    | IdentityCeremony
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityCeremony> | IdentityCeremony),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/passkeys/assertion",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartPasskeySignInResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getFinishPasskeySignInMockHandler = (
  overrideResponse?:
    | IdentitySignedIn
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentitySignedIn> | IdentitySignedIn),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/passkeys/assertion/:ceremony",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getFinishPasskeySignInResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartPasskeyRegistrationMockHandler = (
  overrideResponse?:
    | IdentityCeremony
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityCeremony> | IdentityCeremony),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/passkeys/registration",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartPasskeyRegistrationResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getFinishPasskeyRegistrationMockHandler = (
  overrideResponse?:
    | IdentityPasskey
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityPasskey> | IdentityPasskey),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/passkeys/registration/:ceremony",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getFinishPasskeyRegistrationResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getDeletePasskeyMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/passkeys/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getUpdatePasskeyMockHandler = (
  overrideResponse?:
    | IdentityPasskey
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IdentityPasskey> | IdentityPasskey),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/passkeys/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdatePasskeyResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetResetLinkByTokenMockHandler = (
  overrideResponse?:
    | IdentityLinkView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityLinkView> | IdentityLinkView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/reset-links/by-token/:token",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetResetLinkByTokenResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRedeemResetLinkMockHandler = (
  overrideResponse?:
    | IdentitySignedIn
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentitySignedIn> | IdentitySignedIn),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/reset-links/by-token/:token/pin",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRedeemResetLinkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteSessionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/session",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetSessionMockHandler = (
  overrideResponse?:
    | IdentitySession
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentitySession> | IdentitySession),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/session",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSessionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateSessionMockHandler = (
  overrideResponse?:
    | IdentitySignedIn
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentitySignedIn> | IdentitySignedIn),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/session",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateSessionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getReconfirmSessionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/session/reconfirm",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getStartPasskeyReconfirmMockHandler = (
  overrideResponse?:
    | IdentityCeremony
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityCeremony> | IdentityCeremony),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/session/reconfirm/passkey",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartPasskeyReconfirmResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getFinishPasskeyReconfirmMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/session/reconfirm/passkey/:ceremony",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getEndOtherSessionsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/sessions",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getListSessionsMockHandler = (
  overrideResponse?:
    | IdentitySessionRows
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentitySessionRows> | IdentitySessionRows),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/sessions",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListSessionsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getEndSessionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/sessions/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getClaimControllerMockHandler = (
  overrideResponse?:
    | IdentityClaimed
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityClaimed> | IdentityClaimed),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/setup/controller",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getClaimControllerResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getListSignupLinksMockHandler = (
  overrideResponse?:
    | IdentityPendingLinks
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityPendingLinks> | IdentityPendingLinks),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/signup-links",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListSignupLinksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateSignupLinkMockHandler = (
  overrideResponse?:
    | IdentityIssuedSignupLink
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityIssuedSignupLink> | IdentityIssuedSignupLink),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/signup-links",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateSignupLinkResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getGetSignupLinkByTokenMockHandler = (
  overrideResponse?:
    | IdentityLinkView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentityLinkView> | IdentityLinkView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/signup-links/by-token/:token",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSignupLinkByTokenResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRedeemSignupLinkMockHandler = (
  overrideResponse?:
    | IdentitySignedIn
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentitySignedIn> | IdentitySignedIn),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/signup-links/by-token/:token/account",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRedeemSignupLinkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteSignupLinkMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/signup-links/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetSignupLinkMockHandler = (
  overrideResponse?:
    | IdentitySignupLinkURL
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IdentitySignupLinkURL> | IdentitySignupLinkURL),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/signup-links/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSignupLinkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getResendSignupLinkMockHandler = (
  overrideResponse?:
    | IdentityIssuedSignupLink
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentityIssuedSignupLink> | IdentityIssuedSignupLink),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/signup-links/:id/resend",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getResendSignupLinkResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};
export const getIdentityMock = () => [
  getUpdateOwnAccountMockHandler(),
  getStartEmailChangeMockHandler(),
  getChangePINMockHandler(),
  getListAccountsMockHandler(),
  getDeleteAccountMockHandler(),
  getGetAccountMockHandler(),
  getUpdateAccountMockHandler(),
  getCreateResetLinkMockHandler(),
  getEndAccountSessionsMockHandler(),
  getListAccountSessionsMockHandler(),
  getEndAccountSessionMockHandler(),
  getGetAPIKeyMockHandler(),
  getRegenerateAPIKeyMockHandler(),
  getGetEmailLinkByTokenMockHandler(),
  getConfirmEmailMockHandler(),
  getGetAddressMockHandler(),
  getSetAddressMockHandler(),
  getGetIdentitySettingsMockHandler(),
  getUpdateIdentitySettingsMockHandler(),
  getListPasskeysMockHandler(),
  getStartPasskeySignInMockHandler(),
  getFinishPasskeySignInMockHandler(),
  getStartPasskeyRegistrationMockHandler(),
  getFinishPasskeyRegistrationMockHandler(),
  getDeletePasskeyMockHandler(),
  getUpdatePasskeyMockHandler(),
  getGetResetLinkByTokenMockHandler(),
  getRedeemResetLinkMockHandler(),
  getDeleteSessionMockHandler(),
  getGetSessionMockHandler(),
  getCreateSessionMockHandler(),
  getReconfirmSessionMockHandler(),
  getStartPasskeyReconfirmMockHandler(),
  getFinishPasskeyReconfirmMockHandler(),
  getEndOtherSessionsMockHandler(),
  getListSessionsMockHandler(),
  getEndSessionMockHandler(),
  getClaimControllerMockHandler(),
  getListSignupLinksMockHandler(),
  getCreateSignupLinkMockHandler(),
  getGetSignupLinkByTokenMockHandler(),
  getRedeemSignupLinkMockHandler(),
  getDeleteSignupLinkMockHandler(),
  getGetSignupLinkMockHandler(),
  getResendSignupLinkMockHandler(),
];
