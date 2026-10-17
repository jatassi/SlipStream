import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { MailOutcome, MailSettings } from "../../model";

export const getGetMailSettingsResponseMock = (
  overrideResponse: Partial<Extract<MailSettings, object>> = {},
): MailSettings => ({
  fromAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
  fromName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  host: faker.string.alpha({ length: { min: 10, max: 20 } }),
  passwordSet: faker.datatype.boolean(),
  port: faker.number.int({ min: 1 }),
  security: faker.helpers.arrayElement(["none", "starttls", "tls"] as const),
  username: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getUpdateMailSettingsResponseMock = (
  overrideResponse: Partial<Extract<MailSettings, object>> = {},
): MailSettings => ({
  fromAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
  fromName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  host: faker.string.alpha({ length: { min: 10, max: 20 } }),
  passwordSet: faker.datatype.boolean(),
  port: faker.number.int({ min: 1 }),
  security: faker.helpers.arrayElement(["none", "starttls", "tls"] as const),
  username: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getSendTestMailResponseMock = (
  overrideResponse: Partial<Extract<MailOutcome, object>> = {},
): MailOutcome => ({
  ...(faker.datatype.boolean()
    ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  outcome: faker.helpers.arrayElement(["sent", "failed", "not-set-up"] as const),
  ...overrideResponse,
});

export const getGetMailSettingsMockHandler = (
  overrideResponse?:
    | MailSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MailSettings> | MailSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/mail/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetMailSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateMailSettingsMockHandler = (
  overrideResponse?:
    | MailSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<MailSettings> | MailSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/mail/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateMailSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSendTestMailMockHandler = (
  overrideResponse?:
    | MailOutcome
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MailOutcome> | MailOutcome),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/mail/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSendTestMailResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getMailMock = () => [
  getGetMailSettingsMockHandler(),
  getUpdateMailSettingsMockHandler(),
  getSendTestMailMockHandler(),
];
