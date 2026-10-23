import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { ShellCounts } from "../../model";

export const getGetCountsResponseMock = (
  overrideResponse: Partial<Extract<ShellCounts, object>> = {},
): ShellCounts => ({
  activity: faker.number.int({ min: 0 }),
  alerts: {
    health: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["warn", "fail"] as const),
      null,
    ]),
    nodes: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["warn", "fail"] as const),
      null,
    ]),
    tasks: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["warn", "fail"] as const),
      null,
    ]),
  },
  ...overrideResponse,
});

export const getGetCountsMockHandler = (
  overrideResponse?:
    | ShellCounts
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<ShellCounts> | ShellCounts),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/counts",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetCountsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getShellMock = () => [getGetCountsMockHandler()];
