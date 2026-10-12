import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { IdentityClaimed, IdentitySession } from "../../model";

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
  getDeleteSessionMockHandler(),
  getGetSessionMockHandler(),
  getCreateSessionMockHandler(),
  getClaimControllerMockHandler(),
];
