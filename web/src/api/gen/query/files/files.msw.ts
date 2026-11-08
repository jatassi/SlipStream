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
    code: faker.helpers.arrayElement(["node_away", "client_away"] as const),
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
    codec: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["x264", "x265", "AV1", "VP9", "XviD", "MPEG2"] as const),
      null,
    ]),
    cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        itemId: faker.string.uuid(),
        targetId: faker.string.uuid(),
      }),
    ),
    folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
    goneAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    id: faker.string.uuid(),
    kept: faker.datatype.boolean(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    media: faker.helpers.arrayElement([
      {
        audio: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            atmos: faker.datatype.boolean(),
            channels: faker.number.int({ min: 0 }),
            codec: faker.string.alpha({ length: { min: 10, max: 20 } }),
            default: faker.datatype.boolean(),
            language: faker.helpers.arrayElement([
              faker.string.alpha({ length: { min: 10, max: 20 } }),
              null,
            ]),
            layout: faker.string.alpha({ length: { min: 10, max: 20 } }),
            profile: faker.string.alpha({ length: { min: 10, max: 20 } }),
            title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        bitrate: faker.number.int({ min: 0 }),
        container: faker.string.alpha({ length: { min: 10, max: 20 } }),
        durationSeconds: faker.number.int({ min: 0 }),
        subtitles: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => ({
          codec: faker.string.alpha({ length: { min: 10, max: 20 } }),
          default: faker.datatype.boolean(),
          forced: faker.datatype.boolean(),
          image: faker.datatype.boolean(),
          language: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        })),
        video: {
          bitDepth: faker.number.int({ min: 0 }),
          codec: faker.string.alpha({ length: { min: 10, max: 20 } }),
          dolbyVision: faker.helpers.arrayElement([
            {
              compatibility: faker.number.int({ min: 0 }),
              level: faker.number.int({ min: 0 }),
              profile: faker.number.int({ min: 0 }),
            },
            null,
          ]),
          frameRate: faker.number.float({ min: 0, fractionDigits: 2 }),
          hdr: faker.helpers.arrayElements(["DV", "HDR10+", "HDR10", "HLG", "SDR"] as const),
          height: faker.number.int({ min: 0 }),
          profile: faker.string.alpha({ length: { min: 10, max: 20 } }),
          width: faker.number.int({ min: 0 }),
        },
      },
      null,
    ]),
    modifiedAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    probeError: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
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

export const getForgetFilesMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/files/forget",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

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
export const getFilesMock = () => [
  getForgetFilesMockHandler(),
  getGetWorkCellsMockHandler(),
  getListWorkFilesMockHandler(),
];
