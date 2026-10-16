import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { IdentityClaimed, IdentitySession, IdentitySettings } from "../../model";

export const getGetIdentitySettingsResponseMock = (
  overrideResponse: Partial<Extract<IdentitySettings, object>> = {},
): IdentitySettings => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  addressFailures: faker.number.int(),
  addressWindowSeconds: faker.number.int(),
  emailLinkLifetimeSeconds: faker.number.int(),
  pinOffAfter: faker.number.int(),
  pinWaitAfter: faker.number.int(),
  pinWaitSeconds: faker.number.int(),
  reconfirmWindowSeconds: faker.number.int(),
  resetLinkLifetimeSeconds: faker.number.int(),
  sessionLifetimeSeconds: faker.number.int(),
  signupLinkLifetimeSeconds: faker.number.int(),
  suggestedAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getUpdateIdentitySettingsResponseMock = (
  overrideResponse: Partial<Extract<IdentitySettings, object>> = {},
): IdentitySettings => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  addressFailures: faker.number.int(),
  addressWindowSeconds: faker.number.int(),
  emailLinkLifetimeSeconds: faker.number.int(),
  pinOffAfter: faker.number.int(),
  pinWaitAfter: faker.number.int(),
  pinWaitSeconds: faker.number.int(),
  reconfirmWindowSeconds: faker.number.int(),
  resetLinkLifetimeSeconds: faker.number.int(),
  sessionLifetimeSeconds: faker.number.int(),
  signupLinkLifetimeSeconds: faker.number.int(),
  suggestedAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
  ...overrideResponse,
});

export const getCreateSessionResponseMock = (
  overrideResponse: Partial<Extract<IdentitySession, object>> = {},
): IdentitySession => ({
  account: {
    admin: faker.datatype.boolean(),
    displayName: faker.string.alpha({ length: { min: 1, max: 100 } }),
    email: faker.internet.email(),
    id: faker.string.uuid(),
  },
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
    | IdentitySession
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IdentitySession> | IdentitySession),
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
export const getIdentityMock = () => [
  getGetIdentitySettingsMockHandler(),
  getUpdateIdentitySettingsMockHandler(),
  getDeleteSessionMockHandler(),
  getGetSessionMockHandler(),
  getCreateSessionMockHandler(),
  getClaimControllerMockHandler(),
];
