import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { RootsRoot, RootsRootList } from "../../model";

export const getListRootsResponseMock = (
  overrideResponse: Partial<Extract<RootsRootList, object>> = {},
): RootsRootList => ({
  roots: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    id: faker.string.uuid(),
    nodeId: faker.string.uuid(),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  ...overrideResponse,
});

export const getCreateRootResponseMock = (
  overrideResponse: Partial<Extract<RootsRoot, object>> = {},
): RootsRoot => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  id: faker.string.uuid(),
  nodeId: faker.string.uuid(),
  path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListRootsMockHandler = (
  overrideResponse?:
    | RootsRootList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<RootsRootList> | RootsRootList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/roots",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListRootsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateRootMockHandler = (
  overrideResponse?:
    | RootsRoot
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<RootsRoot> | RootsRoot),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/roots",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateRootResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};
export const getRootsMock = () => [getListRootsMockHandler(), getCreateRootMockHandler()];
