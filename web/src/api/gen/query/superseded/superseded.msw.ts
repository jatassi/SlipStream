import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { SupersededView } from "../../model";

export const getGetSupersededResponseMock = (
  overrideResponse: Partial<Extract<SupersededView, object>> = {},
): SupersededView => ({
  ...(faker.datatype.boolean() ? { at: faker.date.past().toISOString().slice(0, 19) + "Z" } : {}),
  ...(faker.datatype.boolean()
    ? {
        replacement: {
          address: faker.string.alpha({ length: { min: 10, max: 20 } }),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  state: faker.helpers.arrayElement(["none", "unproven", "proven", "stopped", "retired"] as const),
  ...(faker.datatype.boolean()
    ? { worker: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...overrideResponse,
});

export const getGetSupersededMockHandler = (
  overrideResponse?:
    | SupersededView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<SupersededView> | SupersededView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/superseded",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetSupersededResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getResumeSupersededControllerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/superseded/resume",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getStopSupersededControllerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/superseded/stop",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getWipeSupersededControllerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/superseded/wipe",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 202 });
    },
    options,
  );
};
export const getSupersededMock = () => [
  getGetSupersededMockHandler(),
  getResumeSupersededControllerMockHandler(),
  getStopSupersededControllerMockHandler(),
  getWipeSupersededControllerMockHandler(),
];
