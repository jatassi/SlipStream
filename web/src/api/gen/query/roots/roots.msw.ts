import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { RootsRoot, RootsRootList, RootsWorkRoot } from "../../model";

export const getListRootsResponseMock = (
  overrideResponse: Partial<Extract<RootsRootList, object>> = {},
): RootsRootList => ({
  roots: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    default: faker.datatype.boolean(),
    fileBytes: faker.number.int(),
    id: faker.string.uuid(),
    nodeId: faker.string.uuid(),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    works: faker.number.int(),
  })),
  ...overrideResponse,
});

export const getCreateRootResponseMock = (
  overrideResponse: Partial<Extract<RootsRoot, object>> = {},
): RootsRoot => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  default: faker.datatype.boolean(),
  fileBytes: faker.number.int(),
  id: faker.string.uuid(),
  nodeId: faker.string.uuid(),
  path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  works: faker.number.int(),
  ...overrideResponse,
});

export const getSetWorkRootResponseMock = (
  overrideResponse: Partial<Extract<RootsWorkRoot, object>> = {},
): RootsWorkRoot => ({
  rootId: faker.string.uuid(),
  workId: faker.string.uuid(),
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

export const getMakeDefaultRootMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/roots/:rootId/default",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getSetWorkRootMockHandler = (
  overrideResponse?:
    | RootsWorkRoot
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<RootsWorkRoot> | RootsWorkRoot),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/works/:workId",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorkRootResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getRootsMock = () => [
  getListRootsMockHandler(),
  getCreateRootMockHandler(),
  getMakeDefaultRootMockHandler(),
  getSetWorkRootMockHandler(),
];
