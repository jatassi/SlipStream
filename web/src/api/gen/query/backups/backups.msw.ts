import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { BackupsAdded, BackupsDestinationList } from "../../model";

export const getListBackupDestinationsResponseMock = (
  overrideResponse: Partial<Extract<BackupsDestinationList, object>> = {},
): BackupsDestinationList => ({
  destinations: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      copies: {
        copied: faker.number.int(),
        copying: faker.number.int(),
        failed: faker.number.int(),
        waiting: faker.number.int(),
      },
      folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
      newest: faker.helpers.arrayElement([
        {
          backupId: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
          check: {
            at: faker.helpers.arrayElement([
              faker.date.past().toISOString().slice(0, 19) + "Z",
              null,
            ]),
            ...(faker.datatype.boolean()
              ? { cause: faker.string.alpha({ length: { min: 10, max: 20 } }) }
              : {}),
            result: faker.helpers.arrayElement([
              "none",
              "asked",
              "ok",
              "mismatch",
              "unreadable",
              "no_local_file",
            ] as const),
          },
          copiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
        null,
      ]),
      nodeId: faker.string.uuid(),
      removing: faker.datatype.boolean(),
    }),
  ),
  ...overrideResponse,
});

export const getAddBackupDestinationResponseMock = (
  overrideResponse: Partial<Extract<BackupsAdded, object>> = {},
): BackupsAdded => ({
  destination: {
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    copies: {
      copied: faker.number.int(),
      copying: faker.number.int(),
      failed: faker.number.int(),
      waiting: faker.number.int(),
    },
    folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
    id: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
    newest: faker.helpers.arrayElement([
      {
        backupId: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
        check: {
          at: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          ...(faker.datatype.boolean()
            ? { cause: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
          result: faker.helpers.arrayElement([
            "none",
            "asked",
            "ok",
            "mismatch",
            "unreadable",
            "no_local_file",
          ] as const),
        },
        copiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.string.uuid(),
    removing: faker.datatype.boolean(),
  },
  room: faker.helpers.arrayElement([{ free: faker.number.int(), need: faker.number.int() }, null]),
  ...overrideResponse,
});

export const getListBackupDestinationsMockHandler = (
  overrideResponse?:
    | BackupsDestinationList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<BackupsDestinationList> | BackupsDestinationList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/backup-destinations",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListBackupDestinationsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddBackupDestinationMockHandler = (
  overrideResponse?:
    | BackupsAdded
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<BackupsAdded> | BackupsAdded),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/backup-destinations",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddBackupDestinationResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getRemoveBackupDestinationMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/backup-destinations/:destinationId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};
export const getBackupsMock = () => [
  getListBackupDestinationsMockHandler(),
  getAddBackupDestinationMockHandler(),
  getRemoveBackupDestinationMockHandler(),
];
