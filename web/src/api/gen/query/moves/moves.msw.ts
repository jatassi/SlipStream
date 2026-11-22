import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  MovesBatchList,
  MovesBatchPage,
  MovesCanceledMoves,
  MovesMovePreview,
  MovesMoveUnit,
  MovesStartedMoves,
} from "../../model";

export const getListMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesBatchList, object>> = {},
): MovesBatchList => ({
  batches: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      canCancel: faker.datatype.boolean(),
      canMoveBack: faker.datatype.boolean(),
      canRetry: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { canceledAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      counts: {
        [faker.string.alphanumeric(5)]: faker.number.int(),
      },
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      ...(faker.datatype.boolean()
        ? { finishedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      id: faker.string.uuid(),
      origin: faker.helpers.arrayElement([
        "work",
        "bulk",
        "root",
        "drain",
        "client",
        "torrent",
      ] as const),
      total: faker.number.int({ min: 0 }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...(faker.datatype.boolean()
    ? { next: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...overrideResponse,
});

export const getStartMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesStartedMoves, object>> = {},
): MovesStartedMoves => ({
  batchId: faker.string.uuid(),
  dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      workId: faker.string.uuid(),
    }),
  ),
  started: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getPreviewMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesMovePreview, object>> = {},
): MovesMovePreview => ({
  button: faker.string.alpha({ length: { min: 10, max: 20 } }),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  homed: faker.number.int({ min: 0 }),
  movable: faker.number.int({ min: 0 }),
  recommendation: faker.string.alpha({ length: { min: 10, max: 20 } }),
  token: faker.string.alpha({ length: { min: 10, max: 20 } }),
  total: faker.number.int({ min: 0 }),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    bytes: faker.number.int({ min: 0 }),
    from: {
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["work"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    reasons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        check: faker.helpers.arrayElement([
          "space",
          "names",
          "folder",
          "destination",
          "media_servers",
          "waits",
          "moving",
          "probe",
        ] as const),
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        color: faker.helpers.arrayElement(["amber", "red"] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    to: {
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    verdict: faker.helpers.arrayElement(["green", "amber", "red"] as const),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getCancelMoveUnitResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveUnit, object>> = {},
): MovesMoveUnit => ({
  batchId: faker.string.uuid(),
  bytes: faker.number.int({ min: 0 }),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement([
        "source_gone",
        "source_failed",
        "damaged",
        "source_written_off",
        "destination_gone",
        "destination_check_failed",
        "switch_refused",
        "destination_unreachable",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  from: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["work"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  phase: faker.helpers.arrayElement([
    "queued",
    "transfer",
    "place",
    "switch",
    "cleanup",
    "done",
    "failed",
    "canceled",
  ] as const),
  switchNow: faker.datatype.boolean(),
  to: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  undone: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? {
        waiting: faker.helpers.arrayElement([
          "pair_busy",
          "low_priority",
          "node_offline",
          "no_route",
          "space",
          "renames",
          "downloading",
          "importing",
          "open_files",
          "playing",
          "media_servers",
          "guard",
          "linked_torrent",
          "reconciling",
        ] as const),
      }
    : {}),
  ...(faker.datatype.boolean()
    ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getRetryMoveUnitResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveUnit, object>> = {},
): MovesMoveUnit => ({
  batchId: faker.string.uuid(),
  bytes: faker.number.int({ min: 0 }),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement([
        "source_gone",
        "source_failed",
        "damaged",
        "source_written_off",
        "destination_gone",
        "destination_check_failed",
        "switch_refused",
        "destination_unreachable",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  from: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["work"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  phase: faker.helpers.arrayElement([
    "queued",
    "transfer",
    "place",
    "switch",
    "cleanup",
    "done",
    "failed",
    "canceled",
  ] as const),
  switchNow: faker.datatype.boolean(),
  to: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  undone: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? {
        waiting: faker.helpers.arrayElement([
          "pair_busy",
          "low_priority",
          "node_offline",
          "no_route",
          "space",
          "renames",
          "downloading",
          "importing",
          "open_files",
          "playing",
          "media_servers",
          "guard",
          "linked_torrent",
          "reconciling",
        ] as const),
      }
    : {}),
  ...(faker.datatype.boolean()
    ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getSwitchMoveUnitNowResponseMock = (
  overrideResponse: Partial<Extract<MovesMoveUnit, object>> = {},
): MovesMoveUnit => ({
  batchId: faker.string.uuid(),
  bytes: faker.number.int({ min: 0 }),
  contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement([
        "source_gone",
        "source_failed",
        "damaged",
        "source_written_off",
        "destination_gone",
        "destination_check_failed",
        "switch_refused",
        "destination_unreachable",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  from: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["work"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  phase: faker.helpers.arrayElement([
    "queued",
    "transfer",
    "place",
    "switch",
    "cleanup",
    "done",
    "failed",
    "canceled",
  ] as const),
  switchNow: faker.datatype.boolean(),
  to: {
    nodeId: faker.string.uuid(),
    nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
    root: faker.string.alpha({ length: { min: 10, max: 20 } }),
    rootId: faker.string.uuid(),
  },
  undone: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? {
        waiting: faker.helpers.arrayElement([
          "pair_busy",
          "low_priority",
          "node_offline",
          "no_route",
          "space",
          "renames",
          "downloading",
          "importing",
          "open_files",
          "playing",
          "media_servers",
          "guard",
          "linked_torrent",
          "reconciling",
        ] as const),
      }
    : {}),
  ...(faker.datatype.boolean()
    ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  workId: faker.string.uuid(),
  ...overrideResponse,
});

export const getGetMoveResponseMock = (
  overrideResponse: Partial<Extract<MovesBatchPage, object>> = {},
): MovesBatchPage => ({
  batch: {
    canCancel: faker.datatype.boolean(),
    canMoveBack: faker.datatype.boolean(),
    canRetry: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { canceledAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    counts: {
      [faker.string.alphanumeric(5)]: faker.number.int(),
    },
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    ...(faker.datatype.boolean()
      ? { finishedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    id: faker.string.uuid(),
    origin: faker.helpers.arrayElement([
      "work",
      "bulk",
      "root",
      "drain",
      "client",
      "torrent",
    ] as const),
    total: faker.number.int({ min: 0 }),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean() ? { next: faker.number.int() } : {}),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    batchId: faker.string.uuid(),
    bytes: faker.number.int({ min: 0 }),
    contentType: faker.string.alpha({ length: { min: 10, max: 20 } }),
    failure: faker.helpers.arrayElement([
      {
        code: faker.helpers.arrayElement([
          "source_gone",
          "source_failed",
          "damaged",
          "source_written_off",
          "destination_gone",
          "destination_check_failed",
          "switch_refused",
          "destination_unreachable",
        ] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    from: {
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    id: faker.string.uuid(),
    kind: faker.helpers.arrayElement(["work"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    notes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    phase: faker.helpers.arrayElement([
      "queued",
      "transfer",
      "place",
      "switch",
      "cleanup",
      "done",
      "failed",
      "canceled",
    ] as const),
    switchNow: faker.datatype.boolean(),
    to: {
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    undone: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? {
          waiting: faker.helpers.arrayElement([
            "pair_busy",
            "low_priority",
            "node_offline",
            "no_route",
            "space",
            "renames",
            "downloading",
            "importing",
            "open_files",
            "playing",
            "media_servers",
            "guard",
            "linked_torrent",
            "reconciling",
          ] as const),
        }
      : {}),
    ...(faker.datatype.boolean()
      ? { waitingSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
      : {}),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getPreviewMoveBackResponseMock = (
  overrideResponse: Partial<Extract<MovesMovePreview, object>> = {},
): MovesMovePreview => ({
  button: faker.string.alpha({ length: { min: 10, max: 20 } }),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  homed: faker.number.int({ min: 0 }),
  movable: faker.number.int({ min: 0 }),
  recommendation: faker.string.alpha({ length: { min: 10, max: 20 } }),
  token: faker.string.alpha({ length: { min: 10, max: 20 } }),
  total: faker.number.int({ min: 0 }),
  units: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    bytes: faker.number.int({ min: 0 }),
    from: {
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["work"] as const),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    reasons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        check: faker.helpers.arrayElement([
          "space",
          "names",
          "folder",
          "destination",
          "media_servers",
          "waits",
          "moving",
          "probe",
        ] as const),
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        color: faker.helpers.arrayElement(["amber", "red"] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    to: {
      nodeId: faker.string.uuid(),
      nodeName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      root: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootId: faker.string.uuid(),
    },
    verdict: faker.helpers.arrayElement(["green", "amber", "red"] as const),
    workId: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getCancelMovesResponseMock = (
  overrideResponse: Partial<Extract<MovesCanceledMoves, object>> = {},
): MovesCanceledMoves => ({
  canceled: faker.number.int({ min: 0 }),
  switched: faker.number.int({ min: 0 }),
  words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListMovesMockHandler = (
  overrideResponse?:
    | MovesBatchList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesBatchList> | MovesBatchList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMovesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartMovesMockHandler = (
  overrideResponse?:
    | MovesStartedMoves
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesStartedMoves> | MovesStartedMoves),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartMovesResponseMock(),
        { status: 202 },
      );
    },
    options,
  );
};

export const getPreviewMovesMockHandler = (
  overrideResponse?:
    | MovesMovePreview
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMovePreview> | MovesMovePreview),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/preview",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPreviewMovesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelMoveUnitMockHandler = (
  overrideResponse?:
    | MovesMoveUnit
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMoveUnit> | MovesMoveUnit),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/units/:unitId/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelMoveUnitResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRetryMoveUnitMockHandler = (
  overrideResponse?:
    | MovesMoveUnit
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMoveUnit> | MovesMoveUnit),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/units/:unitId/retry",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRetryMoveUnitResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSwitchMoveUnitNowMockHandler = (
  overrideResponse?:
    | MovesMoveUnit
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesMoveUnit> | MovesMoveUnit),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/units/:unitId/switch-now",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSwitchMoveUnitNowResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetMoveMockHandler = (
  overrideResponse?:
    | MovesBatchPage
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesBatchPage> | MovesBatchPage),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves/:batchId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetMoveResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPreviewMoveBackMockHandler = (
  overrideResponse?:
    | MovesMovePreview
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MovesMovePreview> | MovesMovePreview),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/moves/:batchId/back",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPreviewMoveBackResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelMovesMockHandler = (
  overrideResponse?:
    | MovesCanceledMoves
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MovesCanceledMoves> | MovesCanceledMoves),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/moves/:batchId/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelMovesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getMovesMock = () => [
  getListMovesMockHandler(),
  getStartMovesMockHandler(),
  getPreviewMovesMockHandler(),
  getCancelMoveUnitMockHandler(),
  getRetryMoveUnitMockHandler(),
  getSwitchMoveUnitNowMockHandler(),
  getGetMoveMockHandler(),
  getPreviewMoveBackMockHandler(),
  getCancelMovesMockHandler(),
];
