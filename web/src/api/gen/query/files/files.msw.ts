import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  FilesCellList,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesMetCell,
  FilesMissingCell,
} from "../../model";

export const getGetWorkCellsResponseFilesMissingCellMock = (
  overrideResponse: Partial<FilesMissingCell> = {},
): FilesMissingCell => ({
  ...{ state: faker.helpers.arrayElement(["missing"] as const) },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesDownloadingCellMock = (
  overrideResponse: Partial<FilesDownloadingCell> = {},
): FilesDownloadingCell => ({
  ...{ state: faker.helpers.arrayElement(["downloading"] as const) },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesFailedCellMock = (
  overrideResponse: Partial<FilesFailedCell> = {},
): FilesFailedCell => ({
  ...{ state: faker.helpers.arrayElement(["failed"] as const) },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesMetCellMock = (
  overrideResponse: Partial<FilesMetCell> = {},
): FilesMetCell => ({
  ...{
    file: {
      id: faker.string.uuid(),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      probe: { height: faker.number.int({ min: 0 }), width: faker.number.int({ min: 0 }) },
      size: faker.number.int({ min: 0 }),
    },
    state: faker.helpers.arrayElement(["met"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseMock = (
  overrideResponse: Partial<Extract<FilesCellList, object>> = {},
): FilesCellList => ({
  cells: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkCellsResponseFilesMissingCellMock() },
      { ...getGetWorkCellsResponseFilesDownloadingCellMock() },
      { ...getGetWorkCellsResponseFilesFailedCellMock() },
      { ...getGetWorkCellsResponseFilesMetCellMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getGetWorkCellsMockHandler = (
  overrideResponse?:
    | FilesCellList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<FilesCellList> | FilesCellList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/cells",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkCellsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getFilesMock = () => [getGetWorkCellsMockHandler()];
