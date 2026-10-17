import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  FilesCellList,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesMetCell,
  FilesMissingCell,
  FilesUpgradableCell,
} from "../../model";

export const getGetWorkCellsResponseFilesMissingCellMock = (
  overrideResponse: Partial<FilesMissingCell> = {},
): FilesMissingCell => ({
  ...{
    edition: faker.helpers.arrayElement(["standard"] as const),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["missing"] as const),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesDownloadingCellMock = (
  overrideResponse: Partial<FilesDownloadingCell> = {},
): FilesDownloadingCell => ({
  ...{
    edition: faker.helpers.arrayElement(["standard"] as const),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["downloading"] as const),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesFailedCellMock = (
  overrideResponse: Partial<FilesFailedCell> = {},
): FilesFailedCell => ({
  ...{
    edition: faker.helpers.arrayElement(["standard"] as const),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["failed"] as const),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesUpgradableCellMock = (
  overrideResponse: Partial<FilesUpgradableCell> = {},
): FilesUpgradableCell => ({
  ...{
    edition: faker.helpers.arrayElement(["standard"] as const),
    file: {
      id: faker.string.uuid(),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      probe: { height: faker.number.int({ min: 0 }), width: faker.number.int({ min: 0 }) },
      quality: faker.helpers.arrayElement([
        "sdtv",
        "dvd",
        "webrip-480p",
        "hdtv-720p",
        "webrip-720p",
        "webdl-720p",
        "bluray-720p",
        "hdtv-1080p",
        "webrip-1080p",
        "webdl-1080p",
        "bluray-1080p",
        "remux-1080p",
        "hdtv-2160p",
        "webrip-2160p",
        "webdl-2160p",
        "bluray-2160p",
        "remux-2160p",
        "cam",
      ] as const),
      size: faker.number.int({ min: 0 }),
    },
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["upgradable"] as const),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesMetCellMock = (
  overrideResponse: Partial<FilesMetCell> = {},
): FilesMetCell => ({
  ...{
    edition: faker.helpers.arrayElement(["standard"] as const),
    file: {
      id: faker.string.uuid(),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      probe: { height: faker.number.int({ min: 0 }), width: faker.number.int({ min: 0 }) },
      quality: faker.helpers.arrayElement([
        "sdtv",
        "dvd",
        "webrip-480p",
        "hdtv-720p",
        "webrip-720p",
        "webdl-720p",
        "bluray-720p",
        "hdtv-1080p",
        "webrip-1080p",
        "webdl-1080p",
        "bluray-1080p",
        "remux-1080p",
        "hdtv-2160p",
        "webrip-2160p",
        "webdl-2160p",
        "bluray-2160p",
        "remux-2160p",
        "cam",
      ] as const),
      size: faker.number.int({ min: 0 }),
    },
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["met"] as const),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    wanted: faker.datatype.boolean(),
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
      { ...getGetWorkCellsResponseFilesUpgradableCellMock() },
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
