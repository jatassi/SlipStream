import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  RootsListing,
  RootsMadeFolder,
  RootsRoot,
  RootsRootList,
  RootsSuggestions,
  RootsVerdict,
  RootsWorkRoot,
} from "../../model";

export const getBrowseFoldersResponseMock = (
  overrideResponse: Partial<Extract<RootsListing, object>> = {},
): RootsListing => ({
  folders: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      readable: faker.datatype.boolean(),
    }),
  ),
  hidden: faker.number.int(),
  parent: faker.string.alpha({ length: { min: 10, max: 20 } }),
  path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  places: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      free: faker.number.int(),
      ...(faker.datatype.boolean()
        ? { kind: faker.helpers.arrayElement(["local", "removable", "network"] as const) }
        : {}),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean() ? { rootId: faker.string.uuid() } : {}),
      total: faker.number.int(),
    }),
  ),
  segments: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  service: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? {
        volume: {
          free: faker.number.int(),
          id: faker.string.alpha({ length: { min: 10, max: 20 } }),
          kind: faker.helpers.arrayElement(["local", "removable", "network"] as const),
          mount: faker.string.alpha({ length: { min: 10, max: 20 } }),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          sampledAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          total: faker.number.int(),
        },
      }
    : {}),
  ...overrideResponse,
});

export const getMakeFolderResponseMock = (
  overrideResponse: Partial<Extract<RootsMadeFolder, object>> = {},
): RootsMadeFolder => ({
  path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getSuggestFoldersResponseMock = (
  overrideResponse: Partial<Extract<RootsSuggestions, object>> = {},
): RootsSuggestions => ({
  suggestions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

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
    ...(faker.datatype.boolean()
      ? { rehome: { checked: faker.number.int(), total: faker.number.int() } }
      : {}),
    state: faker.helpers.arrayElement([
      "ready",
      "missing",
      "unmounted",
      "foreign",
      "unwritable",
      "unresponsive",
      "checking",
      "offline",
      "written_off",
    ] as const),
    ...(faker.datatype.boolean()
      ? { volumeId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
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
  ...(faker.datatype.boolean()
    ? { rehome: { checked: faker.number.int(), total: faker.number.int() } }
    : {}),
  state: faker.helpers.arrayElement([
    "ready",
    "missing",
    "unmounted",
    "foreign",
    "unwritable",
    "unresponsive",
    "checking",
    "offline",
    "written_off",
  ] as const),
  ...(faker.datatype.boolean()
    ? { volumeId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  works: faker.number.int(),
  ...overrideResponse,
});

export const getCheckRootResponseMock = (
  overrideResponse: Partial<Extract<RootsVerdict, object>> = {},
): RootsVerdict => ({
  path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? {
        refusal: {
          reason: faker.helpers.arrayElement([
            "not_absolute",
            "not_found",
            "not_a_folder",
            "unwritable",
            "inside_root",
            "contains_root",
            "inside_data",
            "contains_data",
            "inside_downloads",
            "contains_downloads",
            "unresponsive",
            "node_offline",
          ] as const),
          ...(faker.datatype.boolean() ? { rootId: faker.string.uuid() } : {}),
        },
      }
    : {}),
  usable: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? {
        volume: {
          free: faker.number.int(),
          id: faker.string.alpha({ length: { min: 10, max: 20 } }),
          kind: faker.helpers.arrayElement(["local", "removable", "network"] as const),
          mount: faker.string.alpha({ length: { min: 10, max: 20 } }),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          sampledAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          total: faker.number.int(),
        },
      }
    : {}),
  ...overrideResponse,
});

export const getRehomeRootResponseMock = (
  overrideResponse: Partial<Extract<RootsRoot, object>> = {},
): RootsRoot => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  default: faker.datatype.boolean(),
  fileBytes: faker.number.int(),
  id: faker.string.uuid(),
  nodeId: faker.string.uuid(),
  path: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { rehome: { checked: faker.number.int(), total: faker.number.int() } }
    : {}),
  state: faker.helpers.arrayElement([
    "ready",
    "missing",
    "unmounted",
    "foreign",
    "unwritable",
    "unresponsive",
    "checking",
    "offline",
    "written_off",
  ] as const),
  ...(faker.datatype.boolean()
    ? { volumeId: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
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

export const getBrowseFoldersMockHandler = (
  overrideResponse?:
    | RootsListing
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<RootsListing> | RootsListing),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/:id/folders",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getBrowseFoldersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getMakeFolderMockHandler = (
  overrideResponse?:
    | RootsMadeFolder
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<RootsMadeFolder> | RootsMadeFolder),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/nodes/:id/folders",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getMakeFolderResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getSuggestFoldersMockHandler = (
  overrideResponse?:
    | RootsSuggestions
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<RootsSuggestions> | RootsSuggestions),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/:id/folders/suggest",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSuggestFoldersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

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

export const getCheckRootMockHandler = (
  overrideResponse?:
    | RootsVerdict
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<RootsVerdict> | RootsVerdict),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/roots/check",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCheckRootResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteRootMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/roots/:rootId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getRehomeRootMockHandler = (
  overrideResponse?:
    | RootsRoot
    | ((info: Parameters<Parameters<typeof http.patch>[1]>[0]) => Promise<RootsRoot> | RootsRoot),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/roots/:rootId",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRehomeRootResponseMock(),
        { status: 200 },
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
  getBrowseFoldersMockHandler(),
  getMakeFolderMockHandler(),
  getSuggestFoldersMockHandler(),
  getListRootsMockHandler(),
  getCreateRootMockHandler(),
  getCheckRootMockHandler(),
  getDeleteRootMockHandler(),
  getRehomeRootMockHandler(),
  getMakeDefaultRootMockHandler(),
  getSetWorkRootMockHandler(),
];
