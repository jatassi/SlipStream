import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { MetadataProvider, MetadataProviderList } from "../../model";

export const getListMetadataProvidersResponseMock = (
  overrideResponse: Partial<Extract<MetadataProviderList, object>> = {},
): MetadataProviderList => ({
  providers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      details: {
        lastError: faker.helpers.arrayElement([
          {
            at: faker.date.past().toISOString().slice(0, 19) + "Z",
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          null,
        ]),
        lastSuccess: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            at: faker.date.past().toISOString().slice(0, 19) + "Z",
            nodeId: faker.string.uuid(),
            reached: faker.datatype.boolean(),
          }),
        ),
      },
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      key: faker.helpers.arrayElement(["none", "bundled", "yours"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      state: faker.helpers.arrayElement([
        "unused",
        "working",
        "key_refused",
        "rate_limited",
        "out_of_allowance",
        "unreachable",
      ] as const),
      summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    }),
  ),
  ...overrideResponse,
});

export const getClearMetadataKeyResponseMock = (
  overrideResponse: Partial<Extract<MetadataProvider, object>> = {},
): MetadataProvider => ({
  details: {
    lastError: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    lastSuccess: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        nodeId: faker.string.uuid(),
        reached: faker.datatype.boolean(),
      }),
    ),
  },
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  key: faker.helpers.arrayElement(["none", "bundled", "yours"] as const),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  state: faker.helpers.arrayElement([
    "unused",
    "working",
    "key_refused",
    "rate_limited",
    "out_of_allowance",
    "unreachable",
  ] as const),
  summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
  until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  ...overrideResponse,
});

export const getSetMetadataKeyResponseMock = (
  overrideResponse: Partial<Extract<MetadataProvider, object>> = {},
): MetadataProvider => ({
  details: {
    lastError: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    lastSuccess: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        nodeId: faker.string.uuid(),
        reached: faker.datatype.boolean(),
      }),
    ),
  },
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  key: faker.helpers.arrayElement(["none", "bundled", "yours"] as const),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  state: faker.helpers.arrayElement([
    "unused",
    "working",
    "key_refused",
    "rate_limited",
    "out_of_allowance",
    "unreachable",
  ] as const),
  summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
  until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  ...overrideResponse,
});

export const getTestMetadataProviderResponseMock = (
  overrideResponse: Partial<Extract<MetadataProvider, object>> = {},
): MetadataProvider => ({
  details: {
    lastError: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    lastSuccess: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        nodeId: faker.string.uuid(),
        reached: faker.datatype.boolean(),
      }),
    ),
  },
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  key: faker.helpers.arrayElement(["none", "bundled", "yours"] as const),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  state: faker.helpers.arrayElement([
    "unused",
    "working",
    "key_refused",
    "rate_limited",
    "out_of_allowance",
    "unreachable",
  ] as const),
  summary: faker.string.alpha({ length: { min: 10, max: 20 } }),
  until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  ...overrideResponse,
});

export const getListMetadataProvidersMockHandler = (
  overrideResponse?:
    | MetadataProviderList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MetadataProviderList> | MetadataProviderList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/metadata/providers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMetadataProvidersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getClearMetadataKeyMockHandler = (
  overrideResponse?:
    | MetadataProvider
    | ((
        info: Parameters<Parameters<typeof http.delete>[1]>[0],
      ) => Promise<MetadataProvider> | MetadataProvider),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/metadata/providers/:provider/key",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getClearMetadataKeyResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetMetadataKeyMockHandler = (
  overrideResponse?:
    | MetadataProvider
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<MetadataProvider> | MetadataProvider),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/metadata/providers/:provider/key",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetMetadataKeyResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getTestMetadataProviderMockHandler = (
  overrideResponse?:
    | MetadataProvider
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MetadataProvider> | MetadataProvider),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/metadata/providers/:provider/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestMetadataProviderResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getMetadataMock = () => [
  getListMetadataProvidersMockHandler(),
  getClearMetadataKeyMockHandler(),
  getSetMetadataKeyMockHandler(),
  getTestMetadataProviderMockHandler(),
];
