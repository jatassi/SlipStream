import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type { RestoreSummary, RestoreUploaded } from "../../model";

export const getUploadRestoreResponseMock = (
  overrideResponse: Partial<Extract<RestoreUploaded, object>> = {},
): RestoreUploaded => ({
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getOpenRestoreResponseMock = (
  overrideResponse: Partial<Extract<RestoreSummary, object>> = {},
): RestoreSummary => ({
  controller: faker.string.alpha({ length: { min: 10, max: 20 } }),
  newMachine: faker.datatype.boolean(),
  nodes: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    })),
    null,
  ]),
  takenAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getUploadRestoreMockHandler = (
  overrideResponse?:
    | RestoreUploaded
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<RestoreUploaded> | RestoreUploaded),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/setup/restore",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUploadRestoreResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getConfirmRestoreMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/setup/restore/:id/confirm",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 202 });
    },
    options,
  );
};

export const getOpenRestoreMockHandler = (
  overrideResponse?:
    | RestoreSummary
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<RestoreSummary> | RestoreSummary),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/setup/restore/:id/open",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getOpenRestoreResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getRestoreMock = () => [
  getUploadRestoreMockHandler(),
  getConfirmRestoreMockHandler(),
  getOpenRestoreMockHandler(),
];
