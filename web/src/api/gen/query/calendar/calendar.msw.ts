import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { CalendarAnswer } from "../../model";

export const getGetCalendarResponseMock = (
  overrideResponse: Partial<Extract<CalendarAnswer, object>> = {},
): CalendarAnswer => ({
  days: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    day: faker.date.past().toISOString().slice(0, 10),
    rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            id: faker.string.uuid(),
            label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        itemsWords: faker.string.alpha({ length: { min: 10, max: 20 } }),
        kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
        kindLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
        met: faker.number.int({ min: 0 }),
        monitored: faker.datatype.boolean(),
        of: faker.number.int({ min: 0 }),
        state: faker.helpers.arrayElement([
          faker.helpers.arrayElement([
            "failed",
            "missing",
            "downloading",
            "converting",
            "upgradable",
            "unreleased",
            "met",
          ] as const),
          null,
        ]),
        work: {
          contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        },
      }),
    ),
  })),
  nextDayAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  sentence: faker.helpers.arrayElement([
    {
      needsYou: faker.datatype.boolean(),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  today: faker.date.past().toISOString().slice(0, 10),
  ...overrideResponse,
});

export const getGetCalendarMockHandler = (
  overrideResponse?:
    | CalendarAnswer
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<CalendarAnswer> | CalendarAnswer),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/calendar",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetCalendarResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getCalendarMock = () => [getGetCalendarMockHandler()];
