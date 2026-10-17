import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { ServingWorker } from "../../model";

export const getGetServingResponseMock = (
  overrideResponse: Partial<Extract<ServingWorker, object>> = {},
): ServingWorker => ({
  controller: faker.string.alpha({ length: { min: 10, max: 20 } }),
  fingerprint: faker.string.alpha({ length: { min: 10, max: 20 } }),
  holding: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  seeding: faker.number.int(),
  ...(faker.datatype.boolean()
    ? { since: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...overrideResponse,
});

export const getGetServingMockHandler = (
  overrideResponse?:
    | ServingWorker
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ServingWorker> | ServingWorker),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/serving",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetServingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getServingMock = () => [getGetServingMockHandler()];
