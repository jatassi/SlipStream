import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  FormatsCustomFormat,
  FormatsFormatChange,
  FormatsFormatDeletion,
  FormatsFormatList,
  FormatsPasted,
  FormatsSchema,
} from "../../model";

export const getListCustomFormatsResponseMock = (
  overrideResponse: Partial<Extract<FormatsFormatList, object>> = {},
): FormatsFormatList => ({
  formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      conditions: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => ({
        fields: {
          [faker.string.alphanumeric(5)]: {},
        },
        kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        negate: faker.datatype.boolean(),
        required: faker.datatype.boolean(),
      })),
      dialect: faker.helpers.arrayElement(["slipstream", "radarr", "sonarr"] as const),
      id: faker.string.uuid(),
      includeWhenRenaming: faker.datatype.boolean(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      scoredBy: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      slowReleases: faker.number.int({ min: 0 }),
    }),
  ),
  matchLimitMilliseconds: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getCreateCustomFormatResponseMock = (
  overrideResponse: Partial<Extract<FormatsCustomFormat, object>> = {},
): FormatsCustomFormat => ({
  conditions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fields: {
        [faker.string.alphanumeric(5)]: {},
      },
      kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      negate: faker.datatype.boolean(),
      required: faker.datatype.boolean(),
    }),
  ),
  dialect: faker.helpers.arrayElement(["slipstream", "radarr", "sonarr"] as const),
  id: faker.string.uuid(),
  includeWhenRenaming: faker.datatype.boolean(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  scoredBy: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  slowReleases: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getPasteCustomFormatsResponseMock = (
  overrideResponse: Partial<Extract<FormatsPasted, object>> = {},
): FormatsPasted => ({
  app: faker.helpers.arrayElement(["radarr", "sonarr"] as const),
  formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      outcome: faker.helpers.arrayElement(["add", "replace", "keep_both"] as const),
      ...(faker.datatype.boolean()
        ? {
            sameName: {
              dialect: faker.helpers.arrayElement(["slipstream", "radarr", "sonarr"] as const),
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      savedAs: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  impact: {
    downloading: faker.number.int({ min: 0 }),
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    renamed: faker.number.int({ min: 0 }),
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getCustomFormatSchemaResponseMock = (
  overrideResponse: Partial<Extract<FormatsSchema, object>> = {},
): FormatsSchema => ({
  dialects: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      dialect: faker.helpers.arrayElement(["slipstream", "radarr", "sonarr"] as const),
      kinds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          fields: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
          pattern: faker.datatype.boolean(),
          values: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            value: faker.helpers.arrayElement([
              faker.number.float({ fractionDigits: 2 }),
              faker.string.alpha({ length: { min: 10, max: 20 } }),
              Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
                () => faker.string.alpha({ length: { min: 10, max: 20 } }),
              ),
            ]),
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          })),
          words: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
    }),
  ),
  ...overrideResponse,
});

export const getDeleteCustomFormatResponseMock = (
  overrideResponse: Partial<Extract<FormatsFormatDeletion, object>> = {},
): FormatsFormatDeletion => ({
  impact: {
    downloading: faker.number.int({ min: 0 }),
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    renamed: faker.number.int({ min: 0 }),
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  scoredBy: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  ...overrideResponse,
});

export const getGetCustomFormatResponseMock = (
  overrideResponse: Partial<Extract<FormatsCustomFormat, object>> = {},
): FormatsCustomFormat => ({
  conditions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fields: {
        [faker.string.alphanumeric(5)]: {},
      },
      kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      negate: faker.datatype.boolean(),
      required: faker.datatype.boolean(),
    }),
  ),
  dialect: faker.helpers.arrayElement(["slipstream", "radarr", "sonarr"] as const),
  id: faker.string.uuid(),
  includeWhenRenaming: faker.datatype.boolean(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  scoredBy: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  slowReleases: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getUpdateCustomFormatResponseMock = (
  overrideResponse: Partial<Extract<FormatsFormatChange, object>> = {},
): FormatsFormatChange => ({
  format: {
    conditions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fields: {
          [faker.string.alphanumeric(5)]: {},
        },
        kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        negate: faker.datatype.boolean(),
        required: faker.datatype.boolean(),
      }),
    ),
    dialect: faker.helpers.arrayElement(["slipstream", "radarr", "sonarr"] as const),
    id: faker.string.uuid(),
    includeWhenRenaming: faker.datatype.boolean(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    scoredBy: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    slowReleases: faker.number.int({ min: 0 }),
  },
  impact: {
    downloading: faker.number.int({ min: 0 }),
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    renamed: faker.number.int({ min: 0 }),
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getListCustomFormatsMockHandler = (
  overrideResponse?:
    | FormatsFormatList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<FormatsFormatList> | FormatsFormatList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/custom-formats",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListCustomFormatsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateCustomFormatMockHandler = (
  overrideResponse?:
    | FormatsCustomFormat
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FormatsCustomFormat> | FormatsCustomFormat),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/custom-formats",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateCustomFormatResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getPasteCustomFormatsMockHandler = (
  overrideResponse?:
    | FormatsPasted
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FormatsPasted> | FormatsPasted),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/custom-formats/paste",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPasteCustomFormatsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCustomFormatSchemaMockHandler = (
  overrideResponse?:
    | FormatsSchema
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<FormatsSchema> | FormatsSchema),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/custom-formats/schema",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCustomFormatSchemaResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteCustomFormatMockHandler = (
  overrideResponse?:
    | FormatsFormatDeletion
    | ((
        info: Parameters<Parameters<typeof http.delete>[1]>[0],
      ) => Promise<FormatsFormatDeletion> | FormatsFormatDeletion),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/custom-formats/:formatId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getDeleteCustomFormatResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetCustomFormatMockHandler = (
  overrideResponse?:
    | FormatsCustomFormat
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<FormatsCustomFormat> | FormatsCustomFormat),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/custom-formats/:formatId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetCustomFormatResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateCustomFormatMockHandler = (
  overrideResponse?:
    | FormatsFormatChange
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<FormatsFormatChange> | FormatsFormatChange),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/custom-formats/:formatId",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateCustomFormatResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getFormatsMock = () => [
  getListCustomFormatsMockHandler(),
  getCreateCustomFormatMockHandler(),
  getPasteCustomFormatsMockHandler(),
  getCustomFormatSchemaMockHandler(),
  getDeleteCustomFormatMockHandler(),
  getGetCustomFormatMockHandler(),
  getUpdateCustomFormatMockHandler(),
];
