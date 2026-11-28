import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  HistoryKindsAnswer,
  HistoryListed,
  HistoryOneEntry,
  HistorySentence,
} from "../../model";

export const getListHistoryResponseMock = (
  overrideResponse: Partial<Extract<HistoryListed, object>> = {},
): HistoryListed => ({
  entries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      group: faker.helpers.arrayElement([
        "grabs",
        "imports",
        "library",
        "subtitles",
        "health",
        "backups",
        "moves",
        "conversions",
      ] as const),
      id: faker.number.int(),
      kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
      links: {
        grab: faker.helpers.arrayElement([faker.string.uuid(), null]),
        health: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        import: faker.helpers.arrayElement([faker.string.uuid(), null]),
      },
      tone: faker.helpers.arrayElement(["info", "good", "warn", "fail"] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      work: faker.helpers.arrayElement([
        {
          contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        },
        null,
      ]),
    }),
  ),
  next: faker.helpers.arrayElement([faker.number.int(), null]),
  ...overrideResponse,
});

export const getListHistoryKindsResponseMock = (
  overrideResponse: Partial<Extract<HistoryKindsAnswer, object>> = {},
): HistoryKindsAnswer => ({
  groups: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      group: faker.helpers.arrayElement([
        "grabs",
        "imports",
        "library",
        "subtitles",
        "health",
        "backups",
        "moves",
        "conversions",
      ] as const),
      kinds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getGetHistorySummaryResponseMock = (): HistorySentence | null =>
  faker.helpers.arrayElement([
    {
      needsYou: faker.number.int({ min: 0, max: 100 }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]);

export const getGetHistoryEntryResponseMock = (
  overrideResponse: Partial<Extract<HistoryOneEntry, object>> = {},
): HistoryOneEntry => ({
  entry: {
    at: faker.date.past().toISOString().slice(0, 19) + "Z",
    group: faker.helpers.arrayElement([
      "grabs",
      "imports",
      "library",
      "subtitles",
      "health",
      "backups",
      "moves",
      "conversions",
    ] as const),
    id: faker.number.int(),
    kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
    links: {
      grab: faker.helpers.arrayElement([faker.string.uuid(), null]),
      health: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      import: faker.helpers.arrayElement([faker.string.uuid(), null]),
    },
    tone: faker.helpers.arrayElement(["info", "good", "warn", "fail"] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    work: faker.helpers.arrayElement([
      {
        contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
        id: faker.string.uuid(),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        posterUrl: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
      },
      null,
    ]),
  },
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    placedBy: faker.string.alpha({ length: { min: 10, max: 20 } }),
    quality: faker.string.alpha({ length: { min: 10, max: 20 } }),
    replaced: faker.string.alpha({ length: { min: 10, max: 20 } }),
    replacedLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  health: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      level: faker.helpers.arrayElement(["warn", "fail"] as const),
      subject: faker.string.alpha({ length: { min: 10, max: 20 } }),
      text: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  lines: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  stages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      stage: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getClearHistoryMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/history",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 202 });
    },
    options,
  );
};

export const getListHistoryMockHandler = (
  overrideResponse?:
    | HistoryListed
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HistoryListed> | HistoryListed),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/history",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListHistoryResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListHistoryKindsMockHandler = (
  overrideResponse?:
    | HistoryKindsAnswer
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HistoryKindsAnswer> | HistoryKindsAnswer),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/history/kinds",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListHistoryKindsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetHistorySummaryMockHandler = (
  overrideResponse?:
    | HistorySentence
    | null
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HistorySentence | null> | HistorySentence | null),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/history/summary",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetHistorySummaryResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetHistoryEntryMockHandler = (
  overrideResponse?:
    | HistoryOneEntry
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HistoryOneEntry> | HistoryOneEntry),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/history/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetHistoryEntryResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getHistoryMock = () => [
  getClearHistoryMockHandler(),
  getListHistoryMockHandler(),
  getListHistoryKindsMockHandler(),
  getGetHistorySummaryMockHandler(),
  getGetHistoryEntryMockHandler(),
];
