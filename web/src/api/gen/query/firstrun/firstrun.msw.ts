import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { FirstrunFirstRun } from "../../model";

export const getGetFirstRunResponseMock = (
  overrideResponse: Partial<Extract<FirstrunFirstRun, object>> = {},
): FirstrunFirstRun => ({
  open: faker.datatype.boolean(),
  steps: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.helpers.arrayElement([
      "address",
      "passkey",
      "media",
      "language",
      "roots",
      "downloads",
      "targets",
    ] as const),
    state: faker.helpers.arrayElement(["todo", "done", "skipped"] as const),
  })),
  ...overrideResponse,
});

export const getSetFirstRunStepResponseMock = (
  overrideResponse: Partial<Extract<FirstrunFirstRun, object>> = {},
): FirstrunFirstRun => ({
  open: faker.datatype.boolean(),
  steps: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.helpers.arrayElement([
      "address",
      "passkey",
      "media",
      "language",
      "roots",
      "downloads",
      "targets",
    ] as const),
    state: faker.helpers.arrayElement(["todo", "done", "skipped"] as const),
  })),
  ...overrideResponse,
});

export const getGetFirstRunMockHandler = (
  overrideResponse?:
    | FirstrunFirstRun
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<FirstrunFirstRun> | FirstrunFirstRun),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/first-run",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetFirstRunResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getFinishFirstRunMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/first-run/finish",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getSetFirstRunStepMockHandler = (
  overrideResponse?:
    | FirstrunFirstRun
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<FirstrunFirstRun> | FirstrunFirstRun),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/first-run/steps/:id",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetFirstRunStepResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCheckSetupCodeMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/setup/code",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};
export const getFirstrunMock = () => [
  getGetFirstRunMockHandler(),
  getFinishFirstRunMockHandler(),
  getSetFirstRunStepMockHandler(),
  getCheckSetupCodeMockHandler(),
];
