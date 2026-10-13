import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  WorksAdded,
  WorksCandidateList,
  WorksContentType,
  WorksContentTypeList,
  WorksDateValue,
  WorksDurationValue,
  WorksNumberValue,
  WorksTextValue,
  WorksTree,
  WorksWorkList,
} from "../../model";

export const getListContentTypesResponseMock = (
  overrideResponse: Partial<Extract<WorksContentTypeList, object>> = {},
): WorksContentTypeList => ({
  contentTypes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      enabled: faker.datatype.boolean(),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          kind: faker.helpers.arrayElement(["text", "number", "date", "duration"] as const),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
        }),
      ),
      id: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      item: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          many: faker.string.alpha({ length: { min: 10, max: 20 } }),
          one: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      work: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
    }),
  ),
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseMock = (
  overrideResponse: Partial<Extract<WorksCandidateList, object>> = {},
): WorksCandidateList => ({
  candidates: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getSearchCatalogResponseWorksTextValueMock() },
            { ...getSearchCatalogResponseWorksNumberValueMock() },
            { ...getSearchCatalogResponseWorksDateValueMock() },
            { ...getSearchCatalogResponseWorksDurationValueMock() },
          ]),
      ),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    }),
  ),
  ...overrideResponse,
});

export const getSetContentTypeEnabledResponseMock = (
  overrideResponse: Partial<Extract<WorksContentType, object>> = {},
): WorksContentType => ({
  enabled: faker.datatype.boolean(),
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement(["text", "number", "date", "duration"] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
    }),
  ),
  id: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  item: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  work: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListWorksResponseMock = (
  overrideResponse: Partial<Extract<WorksWorkList, object>> = {},
): WorksWorkList => ({
  works: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  ...overrideResponse,
});

export const getAddWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksAdded, object>> = {},
): WorksAdded => ({
  created: faker.datatype.boolean(),
  work: {
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getAddWorkResponseWorksTextValueMock() },
        { ...getAddWorkResponseWorksNumberValueMock() },
        { ...getAddWorkResponseWorksDateValueMock() },
        { ...getAddWorkResponseWorksDurationValueMock() },
      ]),
    ),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getAddWorkResponseWorksTextValueMock() },
              { ...getAddWorkResponseWorksNumberValueMock() },
              { ...getAddWorkResponseWorksDateValueMock() },
              { ...getAddWorkResponseWorksDurationValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        position: faker.number.int({ min: 1 }),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        depth: faker.number.int({ min: 1 }),
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getAddWorkResponseWorksTextValueMock() },
              { ...getAddWorkResponseWorksNumberValueMock() },
              { ...getAddWorkResponseWorksDateValueMock() },
              { ...getAddWorkResponseWorksDurationValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        position: faker.number.int({ min: 1 }),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksTree, object>> = {},
): WorksTree => ({
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkResponseWorksTextValueMock() },
      { ...getGetWorkResponseWorksNumberValueMock() },
      { ...getGetWorkResponseWorksDateValueMock() },
      { ...getGetWorkResponseWorksDurationValueMock() },
    ]),
  ),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getGetWorkResponseWorksTextValueMock() },
        { ...getGetWorkResponseWorksNumberValueMock() },
        { ...getGetWorkResponseWorksDateValueMock() },
        { ...getGetWorkResponseWorksDurationValueMock() },
      ]),
    ),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    position: faker.number.int({ min: 1 }),
    withdrawnAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
  })),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      depth: faker.number.int({ min: 1 }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getGetWorkResponseWorksTextValueMock() },
            { ...getGetWorkResponseWorksNumberValueMock() },
            { ...getGetWorkResponseWorksDateValueMock() },
            { ...getGetWorkResponseWorksDurationValueMock() },
          ]),
      ),
      id: faker.string.uuid(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      position: faker.number.int({ min: 1 }),
      withdrawnAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getRefreshWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksTree, object>> = {},
): WorksTree => ({
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getRefreshWorkResponseWorksTextValueMock() },
      { ...getRefreshWorkResponseWorksNumberValueMock() },
      { ...getRefreshWorkResponseWorksDateValueMock() },
      { ...getRefreshWorkResponseWorksDurationValueMock() },
    ]),
  ),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getRefreshWorkResponseWorksTextValueMock() },
        { ...getRefreshWorkResponseWorksNumberValueMock() },
        { ...getRefreshWorkResponseWorksDateValueMock() },
        { ...getRefreshWorkResponseWorksDurationValueMock() },
      ]),
    ),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    position: faker.number.int({ min: 1 }),
    withdrawnAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
  })),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      depth: faker.number.int({ min: 1 }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getRefreshWorkResponseWorksTextValueMock() },
            { ...getRefreshWorkResponseWorksNumberValueMock() },
            { ...getRefreshWorkResponseWorksDateValueMock() },
            { ...getRefreshWorkResponseWorksDurationValueMock() },
          ]),
      ),
      id: faker.string.uuid(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      position: faker.number.int({ min: 1 }),
      withdrawnAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getListContentTypesMockHandler = (
  overrideResponse?:
    | WorksContentTypeList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksContentTypeList> | WorksContentTypeList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListContentTypesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSearchCatalogMockHandler = (
  overrideResponse?:
    | WorksCandidateList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksCandidateList> | WorksCandidateList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types/:contentType/catalog",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSearchCatalogResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetContentTypeEnabledMockHandler = (
  overrideResponse?:
    | WorksContentType
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksContentType> | WorksContentType),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/content-types/:contentType/enabled",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetContentTypeEnabledResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListWorksMockHandler = (
  overrideResponse?:
    | WorksWorkList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksWorkList> | WorksWorkList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListWorksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddWorkMockHandler = (
  overrideResponse?:
    | WorksAdded
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<WorksAdded> | WorksAdded),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetWorkMockHandler = (
  overrideResponse?:
    | WorksTree
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<WorksTree> | WorksTree),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRefreshWorkMockHandler = (
  overrideResponse?:
    | WorksTree
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<WorksTree> | WorksTree),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/:workId/refresh",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRefreshWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getWorksMock = () => [
  getListContentTypesMockHandler(),
  getSearchCatalogMockHandler(),
  getSetContentTypeEnabledMockHandler(),
  getListWorksMockHandler(),
  getAddWorkMockHandler(),
  getGetWorkMockHandler(),
  getRefreshWorkMockHandler(),
];
