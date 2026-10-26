import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  FilesCellList,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesFailedNeed,
  FilesFileList,
  FilesMetCell,
  FilesMissingCell,
  FilesNoRootNeed,
  FilesUnreleasedCell,
  FilesUpgradableCell,
  FilesUpgradeNeed,
  FilesWaitingNeed,
} from "../../model";

export const getGetWorkCellsResponseFilesUnreleasedCellMock = (
  overrideResponse: Partial<FilesUnreleasedCell> = {},
): FilesUnreleasedCell => ({
  ...{
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["unreleased"] as const),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesMissingCellMock = (
  overrideResponse: Partial<FilesMissingCell> = {},
): FilesMissingCell => ({
  ...{
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
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
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
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

export const getGetWorkCellsResponseFilesFailedNeedMock = (
  overrideResponse: Partial<FilesFailedNeed> = {},
): FilesFailedNeed => ({
  ...{
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    grabId: faker.string.uuid(),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["failed"] as const),
    targetId: faker.string.uuid(),
    waiting: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesUpgradeNeedMock = (
  overrideResponse: Partial<FilesUpgradeNeed> = {},
): FilesUpgradeNeed => ({
  ...{
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["upgrade"] as const),
    upgrades: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesWaitingNeedMock = (
  overrideResponse: Partial<FilesWaitingNeed> = {},
): FilesWaitingNeed => ({
  ...{
    code: faker.string.alpha({ length: { min: 10, max: 20 } }),
    grabId: faker.string.uuid(),
    kind: faker.helpers.arrayElement(["waiting"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesNoRootNeedMock = (
  overrideResponse: Partial<FilesNoRootNeed> = {},
): FilesNoRootNeed => ({
  ...{ kind: faker.helpers.arrayElement(["no_root"] as const) },
  ...overrideResponse,
});

export const getGetWorkCellsResponseMock = (
  overrideResponse: Partial<Extract<FilesCellList, object>> = {},
): FilesCellList => ({
  cells: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkCellsResponseFilesUnreleasedCellMock() },
      { ...getGetWorkCellsResponseFilesMissingCellMock() },
      { ...getGetWorkCellsResponseFilesDownloadingCellMock() },
      { ...getGetWorkCellsResponseFilesFailedCellMock() },
      { ...getGetWorkCellsResponseFilesUpgradableCellMock() },
      { ...getGetWorkCellsResponseFilesMetCellMock() },
    ]),
  ),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    itemId: faker.string.uuid(),
    status: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "unreleased",
        "missing",
        "downloading",
        "failed",
        "upgradable",
        "met",
      ] as const),
      null,
    ]),
  })),
  needsYou: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkCellsResponseFilesFailedNeedMock() },
      { ...getGetWorkCellsResponseFilesUpgradeNeedMock() },
      { ...getGetWorkCellsResponseFilesWaitingNeedMock() },
      { ...getGetWorkCellsResponseFilesNoRootNeedMock() },
    ]),
  ),
  summary: {
    cells: {
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    itemStatus: {
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    items: faker.number.int({ min: 0 }),
    status: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "unreleased",
        "missing",
        "downloading",
        "failed",
        "upgradable",
        "met",
      ] as const),
      null,
    ]),
  },
  ...overrideResponse,
});

export const getListWorkFilesResponseMock = (
  overrideResponse: Partial<Extract<FilesFileList, object>> = {},
): FilesFileList => ({
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        itemId: faker.string.uuid(),
        targetId: faker.string.uuid(),
      }),
    ),
    folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
    id: faker.string.uuid(),
    kept: faker.datatype.boolean(),
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
  })),
  folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
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

export const getListWorkFilesMockHandler = (
  overrideResponse?:
    | FilesFileList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<FilesFileList> | FilesFileList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/files",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListWorkFilesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getFilesMock = () => [getGetWorkCellsMockHandler(), getListWorkFilesMockHandler()];
