import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { IndexersIndexer, IndexersIndexerList } from "../../model";

export const getListIndexersResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerList, object>> = {},
): IndexersIndexerList => ({
  indexers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      homeNodeId: faker.string.uuid(),
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      protocol: faker.helpers.arrayElement(["torznab"] as const),
      searchModes: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => ({
        mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
        params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      })),
      url: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getCreateIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  protocol: faker.helpers.arrayElement(["torznab"] as const),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  url: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListIndexersMockHandler = (
  overrideResponse?:
    | IndexersIndexerList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexerList> | IndexersIndexerList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListIndexersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateIndexerResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};
export const getIndexersMock = () => [getListIndexersMockHandler(), getCreateIndexerMockHandler()];
