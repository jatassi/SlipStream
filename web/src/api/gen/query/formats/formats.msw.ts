import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  FormatsCustomFormat,
  FormatsFormatChange,
  FormatsFormatDeletion,
  FormatsFormatList,
  FormatsParseResults,
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
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
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
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
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
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getDuplicateCustomFormatResponseMock = (
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

export const getParseReleasesResponseMock = (
  overrideResponse: Partial<Extract<FormatsParseResults, object>> = {},
): FormatsParseResults => ({
  results: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      covers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      facts: {
        audio: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        channels: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
        edition: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        group: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        hdr: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        ids: {
          [faker.string.alphanumeric(5)]: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
        languages: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        marks: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        modifier: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        quality: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        resolution: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        revision: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        source: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        span: faker.helpers.arrayElement(["", "one", "several", "level", "levels"] as const),
        titles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        video: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        year: faker.helpers.arrayElement([faker.number.int(), null]),
      },
      formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          conditions: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            matched: faker.datatype.boolean(),
            name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            negate: faker.datatype.boolean(),
            required: faker.datatype.boolean(),
            timedOut: faker.datatype.boolean(),
          })),
          draft: faker.datatype.boolean(),
          id: faker.string.uuid(),
          matched: faker.datatype.boolean(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          timedOut: faker.datatype.boolean(),
        }),
      ),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
      targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
          reasons: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            ...(faker.datatype.boolean()
              ? {
                  attribute: faker.helpers.arrayElement([
                    "hdr",
                    "video_codec",
                    "audio_codec",
                    "channels",
                    "language",
                  ] as const),
                }
              : {}),
            code: faker.helpers.arrayElement([
              "wrong_work",
              "wrong_item",
              "wrong_title",
              "cut_not_wanted",
              "quality_not_in_target",
              "filter",
              "size_below",
              "size_above",
              "size_not_checked",
              "score_below_minimum",
              "not_an_upgrade",
              "repack_other_group",
              "blocklisted",
            ] as const),
            soft: faker.datatype.boolean(),
            ...(faker.datatype.boolean() ? { targetId: faker.string.uuid() } : {}),
            ...(faker.datatype.boolean()
              ? { value: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            words: faker.string.alpha({ length: { min: 10, max: 20 } }),
          })),
          score: faker.number.int(),
          targetId: faker.string.uuid(),
          wouldFill: faker.datatype.boolean(),
        }),
      ),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      work: faker.helpers.arrayElement([
        { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
        null,
      ]),
      works: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
    }),
  ),
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

export const getDuplicateCustomFormatMockHandler = (
  overrideResponse?:
    | FormatsCustomFormat
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FormatsCustomFormat> | FormatsCustomFormat),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/custom-formats/:formatId/duplicate",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getDuplicateCustomFormatResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getParseReleasesMockHandler = (
  overrideResponse?:
    | FormatsParseResults
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FormatsParseResults> | FormatsParseResults),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/releases/parse",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getParseReleasesResponseMock(),
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
  getDuplicateCustomFormatMockHandler(),
  getParseReleasesMockHandler(),
];
