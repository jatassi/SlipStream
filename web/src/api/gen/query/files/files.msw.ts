import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  FilesCellList,
  FilesConversionFailedNeed,
  FilesConversionFailure,
  FilesConvertingCell,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesFailedNeed,
  FilesFileList,
  FilesGrabFailure,
  FilesMetCell,
  FilesMissingCell,
  FilesNoRootNeed,
  FilesNotFoundNeed,
  FilesRenamePlan,
  FilesRenameResult,
  FilesUnreleasedCell,
  FilesUpgradableCell,
  FilesUpgradeNeed,
  FilesWaitingNeed,
  FilesWorkRenamePlan,
  MovingOutputBody,
  UnmatchedOutputBody,
} from "../../model";

export const getRenameFilesResponseMock = (
  overrideResponse: Partial<Extract<FilesRenameResult, object>> = {},
): FilesRenameResult => ({
  changed: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  queued: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getCountMovingResponseMock = (
  overrideResponse: Partial<Extract<MovingOutputBody, object>> = {},
): MovingOutputBody => ({ files: faker.number.int({ min: 0 }), ...overrideResponse });

export const getPlanRenamesResponseMock = (
  overrideResponse: Partial<Extract<FilesRenamePlan, object>> = {},
): FilesRenamePlan => ({
  blocked: faker.number.int({ min: 0 }),
  files: faker.number.int({ min: 0 }),
  works: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    blocked: faker.number.int({ min: 0 }),
    contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
    digest: faker.string.alpha({ length: { min: 10, max: 20 } }),
    moving: faker.number.int({ min: 0 }),
    rename: faker.number.int({ min: 0 }),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getPlanWorkRenamesResponseMock = (
  overrideResponse: Partial<Extract<FilesWorkRenamePlan, object>> = {},
): FilesWorkRenamePlan => ({
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  digest: faker.string.alpha({ length: { min: 10, max: 20 } }),
  moving: faker.number.int({ min: 0 }),
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    ...(faker.datatype.boolean()
      ? {
          blocked: {
            detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
            ...(faker.datatype.boolean() ? { otherWorkId: faker.string.uuid() } : {}),
            reason: faker.helpers.arrayElement(["occupied", "too_long", "clash"] as const),
          },
        }
      : {}),
    companions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    fileId: faker.string.uuid(),
    from: faker.string.alpha({ length: { min: 10, max: 20 } }),
    group: faker.number.int({ min: 0 }),
    to: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  title: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getCountUnmatchedResponseMock = (
  overrideResponse: Partial<Extract<UnmatchedOutputBody, object>> = {},
): UnmatchedOutputBody => ({
  contentTypes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
      files: faker.number.int({ min: 1 }),
    }),
  ),
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesUnreleasedCellMock = (
  overrideResponse: Partial<FilesUnreleasedCell> = {},
): FilesUnreleasedCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    gone: faker.datatype.boolean(),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["unreleased"] as const),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesMissingCellMock = (
  overrideResponse: Partial<FilesMissingCell> = {},
): FilesMissingCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    gone: faker.datatype.boolean(),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["missing"] as const),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesDownloadingCellMock = (
  overrideResponse: Partial<FilesDownloadingCell> = {},
): FilesDownloadingCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    file: faker.helpers.arrayElement([
      {
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
      null,
    ]),
    from: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    gone: faker.datatype.boolean(),
    grabId: faker.string.uuid(),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["downloading"] as const),
    step: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "fetching",
        "downloading",
        "reading",
        "sending",
        "placing",
        "waiting",
      ] as const),
      null,
    ]),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesConvertingCellMock = (
  overrideResponse: Partial<FilesConvertingCell> = {},
): FilesConvertingCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
    conversionId: faker.string.uuid(),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    file: faker.helpers.arrayElement([
      {
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
      null,
    ]),
    gone: faker.datatype.boolean(),
    itemId: faker.string.uuid(),
    node: faker.helpers.arrayElement([
      { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      null,
    ]),
    stage: faker.helpers.arrayElement([
      "queued",
      "copying",
      "ready",
      "running",
      "paused",
      "checking",
      "returning",
      "landing",
    ] as const),
    state: faker.helpers.arrayElement(["converting"] as const),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesGrabFailureMock = (
  overrideResponse: Partial<FilesGrabFailure> = {},
): FilesGrabFailure => ({
  ...{ grabId: faker.string.uuid(), kind: faker.helpers.arrayElement(["grab"] as const) },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesConversionFailureMock = (
  overrideResponse: Partial<FilesConversionFailure> = {},
): FilesConversionFailure => ({
  ...{
    code: faker.string.alpha({ length: { min: 10, max: 20 } }),
    conversionId: faker.string.uuid(),
    kind: faker.helpers.arrayElement(["conversion"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesFailedCellMock = (
  overrideResponse: Partial<FilesFailedCell> = {},
): FilesFailedCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
    cause: faker.helpers.arrayElement([
      { ...getGetWorkCellsResponseFilesGrabFailureMock() },
      { ...getGetWorkCellsResponseFilesConversionFailureMock() },
    ]),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    file: faker.helpers.arrayElement([
      {
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
      null,
    ]),
    gone: faker.datatype.boolean(),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["failed"] as const),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesUpgradableCellMock = (
  overrideResponse: Partial<FilesUpgradableCell> = {},
): FilesUpgradableCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
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
    gone: faker.datatype.boolean(),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["upgradable"] as const),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    wanted: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesMetCellMock = (
  overrideResponse: Partial<FilesMetCell> = {},
): FilesMetCell => ({
  ...{
    candidates: faker.number.int({ min: 0 }),
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
    gone: faker.datatype.boolean(),
    held: faker.helpers.arrayElement([
      {
        from: faker.date.past().toISOString().slice(0, 19) + "Z",
        grabId: faker.string.uuid(),
        step: faker.helpers.arrayElement([
          "fetching",
          "downloading",
          "reading",
          "sending",
          "placing",
          "waiting",
        ] as const),
        until: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      },
      null,
    ]),
    itemId: faker.string.uuid(),
    state: faker.helpers.arrayElement(["met"] as const),
    stopped: faker.datatype.boolean(),
    targetId: faker.string.uuid(),
    targetName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    waitUntil: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
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

export const getGetWorkCellsResponseFilesConversionFailedNeedMock = (
  overrideResponse: Partial<FilesConversionFailedNeed> = {},
): FilesConversionFailedNeed => ({
  ...{
    class: faker.helpers.arrayElement(["run", "source"] as const),
    code: faker.string.alpha({ length: { min: 10, max: 20 } }),
    conversionId: faker.string.uuid(),
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["conversion_failed"] as const),
    targetId: faker.string.uuid(),
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

export const getGetWorkCellsResponseFilesNotFoundNeedMock = (
  overrideResponse: Partial<FilesNotFoundNeed> = {},
): FilesNotFoundNeed => ({
  ...{
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    groupKey: faker.string.alpha({ length: { min: 10, max: 20 } }),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["not_found"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkCellsResponseFilesWaitingNeedMock = (
  overrideResponse: Partial<FilesWaitingNeed> = {},
): FilesWaitingNeed => ({
  ...{
    clientError: faker.helpers.arrayElement([
      {
        client: faker.string.alpha({ length: { min: 10, max: 20 } }),
        failsAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        message: faker.string.alpha({ length: { min: 10, max: 20 } }),
        node: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    code: faker.helpers.arrayElement([
      "node_away",
      "client_away",
      "client_off",
      "no_space",
      "root_not_ready",
      "probe_timeout",
      "probe_error",
      "client_error",
    ] as const),
    grabId: faker.string.uuid(),
    importWait: faker.helpers.arrayElement([
      {
        rootId: faker.string.uuid(),
        space: faker.helpers.arrayElement([
          { free: faker.number.int({ min: 0 }), needs: faker.number.int({ min: 1 }) },
          null,
        ]),
      },
      null,
    ]),
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
      { ...getGetWorkCellsResponseFilesConvertingCellMock() },
      { ...getGetWorkCellsResponseFilesFailedCellMock() },
      { ...getGetWorkCellsResponseFilesUpgradableCellMock() },
      { ...getGetWorkCellsResponseFilesMetCellMock() },
    ]),
  ),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    itemId: faker.string.uuid(),
    released: faker.datatype.boolean(),
    status: faker.helpers.arrayElement([
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
  })),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      cells: {
        ...(faker.datatype.boolean() ? { converting: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
      },
      itemStatus: {
        ...(faker.datatype.boolean() ? { converting: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
      },
      items: faker.number.int({ min: 0 }),
      levelId: faker.string.uuid(),
      withdrawn: {
        ...(faker.datatype.boolean() ? { converting: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
        ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
      },
    }),
  ),
  needsYou: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkCellsResponseFilesFailedNeedMock() },
      { ...getGetWorkCellsResponseFilesConversionFailedNeedMock() },
      { ...getGetWorkCellsResponseFilesUpgradeNeedMock() },
      { ...getGetWorkCellsResponseFilesNotFoundNeedMock() },
      { ...getGetWorkCellsResponseFilesWaitingNeedMock() },
      { ...getGetWorkCellsResponseFilesNoRootNeedMock() },
    ]),
  ),
  summary: {
    cells: {
      ...(faker.datatype.boolean() ? { converting: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    itemStatus: {
      ...(faker.datatype.boolean() ? { converting: faker.number.int({ min: 1 }) } : {}),
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
    grab: faker.helpers.arrayElement([
      { id: faker.string.uuid(), release: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      null,
    ]),
    id: faker.string.uuid(),
    kept: faker.datatype.boolean(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    madeFrom: faker.helpers.arrayElement([
      {
        fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
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
    placedBy: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "link",
        "clone",
        "copy",
        "transfer",
        "move",
        "in_place",
        "converted",
      ] as const),
      null,
    ]),
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

export const getRenameFilesMockHandler = (
  overrideResponse?:
    | FilesRenameResult
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FilesRenameResult> | FilesRenameResult),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/renames",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRenameFilesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCountMovingMockHandler = (
  overrideResponse?:
    | MovingOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovingOutputBody> | MovingOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/renames/moving",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCountMovingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPlanRenamesMockHandler = (
  overrideResponse?:
    | FilesRenamePlan
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FilesRenamePlan> | FilesRenamePlan),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/renames/plan",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPlanRenamesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPlanWorkRenamesMockHandler = (
  overrideResponse?:
    | FilesWorkRenamePlan
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<FilesWorkRenamePlan> | FilesWorkRenamePlan),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/renames/plan/:workId",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPlanWorkRenamesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCountUnmatchedMockHandler = (
  overrideResponse?:
    | UnmatchedOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<UnmatchedOutputBody> | UnmatchedOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/renames/unmatched",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCountUnmatchedResponseMock(),
        { status: 200 },
      );
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
  getRenameFilesMockHandler(),
  getCountMovingMockHandler(),
  getPlanRenamesMockHandler(),
  getPlanWorkRenamesMockHandler(),
  getCountUnmatchedMockHandler(),
  getGetWorkCellsMockHandler(),
  getListWorkFilesMockHandler(),
];
