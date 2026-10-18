import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  NodesCodeState,
  NodesDrains,
  NodesGains,
  NodesHandedBack,
  NodesJoinCode,
  NodesLeaves,
  NodesLetsGo,
  NodesMoves,
  NodesNode,
  NodesNodeList,
  NodesOpenCodes,
  NodesOutcome,
  NodesPinned,
  NodesRoleBoard,
  NodesSettings,
  NodesWritesOff,
} from "../../model";

export const getListJoinCodesResponseMock = (
  overrideResponse: Partial<Extract<NodesOpenCodes, object>> = {},
): NodesOpenCodes => ({
  codes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    id: faker.string.uuid(),
  })),
  ...overrideResponse,
});

export const getCreateJoinCodeResponseMock = (
  overrideResponse: Partial<Extract<NodesJoinCode, object>> = {},
): NodesJoinCode => ({
  code: faker.string.alpha({ length: { min: 10, max: 20 } }),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  id: faker.string.uuid(),
  ...overrideResponse,
});

export const getGetJoinCodeResponseMock = (
  overrideResponse: Partial<Extract<NodesCodeState, object>> = {},
): NodesCodeState => ({
  ...(faker.datatype.boolean()
    ? {
        dialing: {
          address: faker.string.alpha({ length: { min: 10, max: 20 } }),
          since: faker.date.past().toISOString().slice(0, 19) + "Z",
          unreachable: faker.datatype.boolean(),
        },
      }
    : {}),
  ...(faker.datatype.boolean() ? { nodeId: faker.string.uuid() } : {}),
  state: faker.helpers.arrayElement(["open", "used", "expired"] as const),
  ...overrideResponse,
});

export const getListNodesResponseMock = (
  overrideResponse: Partial<Extract<NodesNodeList, object>> = {},
): NodesNodeList => ({
  nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    dial: {
      ...(faker.datatype.boolean()
        ? { by: faker.helpers.arrayElement(["node", "controller"] as const) }
        : {}),
      controllerDials: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
    },
    ...(faker.datatype.boolean()
      ? {
          facts: {
            appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
            assigned: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            httpAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
            restarting: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
            running: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          },
        }
      : {}),
    id: faker.string.uuid(),
    intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    link: {
      ...(faker.datatype.boolean()
        ? {
            build: {
              arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
              installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
              os: faker.string.alpha({ length: { min: 10, max: 20 } }),
              version: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      since: faker.date.past().toISOString().slice(0, 19) + "Z",
      state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
    },
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    probes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        fit: faker.helpers.arrayElement(["fits", "weak", "fails"] as const),
        probe: faker.helpers.arrayElement([
          "engine-starts",
          "data-writable",
          "listener-up",
        ] as const),
        ...(faker.datatype.boolean()
          ? {
              reason: faker.helpers.arrayElement([
                "engine_not_listening",
                "downloads_not_writable",
                "downloads_on_network_share",
                "data_not_writable",
                "http_not_listening",
              ] as const),
            }
          : {}),
      }),
    ),
    roles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        ...(faker.datatype.boolean()
          ? {
              held: {
                bytes: faker.number.int(),
                noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
                units: faker.number.int(),
              },
            }
          : {}),
        role: faker.helpers.arrayElement([
          "searching",
          "downloading",
          "storage",
          "converting",
          "subtitles",
          "metadata",
          "media-server-link",
          "notifying",
          "relaying",
          "serving",
        ] as const),
        state: faker.helpers.arrayElement([
          "holding",
          "away",
          "lapsed",
          "draining",
          "writing-off",
          "covering",
        ] as const),
      }),
    ),
    self: faker.datatype.boolean(),
  })),
  ...overrideResponse,
});

export const getGetRoleBoardResponseMock = (
  overrideResponse: Partial<Extract<NodesRoleBoard, object>> = {},
): NodesRoleBoard => ({
  basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
  gestures: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { from: faker.string.uuid() } : {}),
      ...(faker.datatype.boolean()
        ? { note: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      role: faker.helpers.arrayElement([
        "searching",
        "downloading",
        "storage",
        "converting",
        "subtitles",
        "metadata",
        "media-server-link",
        "notifying",
        "relaying",
        "serving",
      ] as const),
      to: faker.string.uuid(),
      verdict: faker.helpers.arrayElement(["allow", "warn", "block"] as const),
      word: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    assigned: faker.helpers.arrayElements([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    away: faker.datatype.boolean(),
    controller: faker.datatype.boolean(),
    id: faker.string.uuid(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    probes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        fit: faker.helpers.arrayElement(["fits", "weak", "fails"] as const),
        probe: faker.helpers.arrayElement([
          "engine-starts",
          "data-writable",
          "listener-up",
        ] as const),
        ...(faker.datatype.boolean()
          ? {
              reason: faker.helpers.arrayElement([
                "engine_not_listening",
                "downloads_not_writable",
                "downloads_on_network_share",
                "data_not_writable",
                "http_not_listening",
              ] as const),
            }
          : {}),
      }),
    ),
    roles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        ...(faker.datatype.boolean()
          ? {
              held: {
                bytes: faker.number.int(),
                noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
                units: faker.number.int(),
              },
            }
          : {}),
        role: faker.helpers.arrayElement([
          "searching",
          "downloading",
          "storage",
          "converting",
          "subtitles",
          "metadata",
          "media-server-link",
          "notifying",
          "relaying",
          "serving",
        ] as const),
        state: faker.helpers.arrayElement([
          "holding",
          "away",
          "lapsed",
          "draining",
          "writing-off",
          "covering",
        ] as const),
      }),
    ),
    suggested: faker.helpers.arrayElements([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
  })),
  ...overrideResponse,
});

export const getSetRolesResponseNodesGainsMock = (
  overrideResponse: Partial<NodesGains> = {},
): NodesGains => ({
  ...{
    kind: faker.helpers.arrayElement(["gains"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesLetsGoMock = (
  overrideResponse: Partial<NodesLetsGo> = {},
): NodesLetsGo => ({
  ...{
    kind: faker.helpers.arrayElement(["lets-go"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesDrainsMock = (
  overrideResponse: Partial<NodesDrains> = {},
): NodesDrains => ({
  ...{
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["drains"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesHandedBackMock = (
  overrideResponse: Partial<NodesHandedBack> = {},
): NodesHandedBack => ({
  ...{
    from: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    kind: faker.helpers.arrayElement(["handed-back"] as const),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    to: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesPinnedMock = (
  overrideResponse: Partial<NodesPinned> = {},
): NodesPinned => ({
  ...{
    kind: faker.helpers.arrayElement(["pinned"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesMovesMock = (
  overrideResponse: Partial<NodesMoves> = {},
): NodesMoves => ({
  ...{
    from: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["moves"] as const),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    to: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesWritesOffMock = (
  overrideResponse: Partial<NodesWritesOff> = {},
): NodesWritesOff => ({
  ...{
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["writes-off"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseNodesLeavesMock = (
  overrideResponse: Partial<NodesLeaves> = {},
): NodesLeaves => ({
  ...{
    kind: faker.helpers.arrayElement(["leaves"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetRolesResponseMock = (
  overrideResponse: Partial<Extract<NodesOutcome, object>> = {},
): NodesOutcome => ({
  basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
  checks: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      reason: faker.helpers.arrayElement([
        "role_taken",
        "probe_fails",
        "probe_weak",
        "offline_unchecked",
        "reach_failed",
        "reached",
        "nothing_to_test",
        "node_is_controller",
        "node_holds_work",
        "nothing_held",
        "keeps_role",
      ] as const),
      ...(faker.datatype.boolean()
        ? {
            role: faker.helpers.arrayElement([
              "searching",
              "downloading",
              "storage",
              "converting",
              "subtitles",
              "metadata",
              "media-server-link",
              "notifying",
              "relaying",
              "serving",
            ] as const),
          }
        : {}),
      verdict: faker.helpers.arrayElement(["allow", "warn", "block"] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  effects: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getSetRolesResponseNodesGainsMock() },
      { ...getSetRolesResponseNodesLetsGoMock() },
      { ...getSetRolesResponseNodesDrainsMock() },
      { ...getSetRolesResponseNodesHandedBackMock() },
      { ...getSetRolesResponseNodesPinnedMock() },
      { ...getSetRolesResponseNodesMovesMock() },
      { ...getSetRolesResponseNodesWritesOffMock() },
      { ...getSetRolesResponseNodesLeavesMock() },
    ]),
  ),
  forfeit: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      nodeId: faker.string.uuid(),
      roles: faker.helpers.arrayElements([
        "searching",
        "downloading",
        "storage",
        "converting",
        "subtitles",
        "metadata",
        "media-server-link",
        "notifying",
        "relaying",
        "serving",
      ] as const),
    }),
  ),
  next: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    nodeId: faker.string.uuid(),
    roles: faker.helpers.arrayElements([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
  })),
  ...overrideResponse,
});

export const getPlanRolesResponseNodesGainsMock = (
  overrideResponse: Partial<NodesGains> = {},
): NodesGains => ({
  ...{
    kind: faker.helpers.arrayElement(["gains"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesLetsGoMock = (
  overrideResponse: Partial<NodesLetsGo> = {},
): NodesLetsGo => ({
  ...{
    kind: faker.helpers.arrayElement(["lets-go"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesDrainsMock = (
  overrideResponse: Partial<NodesDrains> = {},
): NodesDrains => ({
  ...{
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["drains"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesHandedBackMock = (
  overrideResponse: Partial<NodesHandedBack> = {},
): NodesHandedBack => ({
  ...{
    from: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    kind: faker.helpers.arrayElement(["handed-back"] as const),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    to: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesPinnedMock = (
  overrideResponse: Partial<NodesPinned> = {},
): NodesPinned => ({
  ...{
    kind: faker.helpers.arrayElement(["pinned"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesMovesMock = (
  overrideResponse: Partial<NodesMoves> = {},
): NodesMoves => ({
  ...{
    from: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["moves"] as const),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    to: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesWritesOffMock = (
  overrideResponse: Partial<NodesWritesOff> = {},
): NodesWritesOff => ({
  ...{
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["writes-off"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseNodesLeavesMock = (
  overrideResponse: Partial<NodesLeaves> = {},
): NodesLeaves => ({
  ...{
    kind: faker.helpers.arrayElement(["leaves"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPlanRolesResponseMock = (
  overrideResponse: Partial<Extract<NodesOutcome, object>> = {},
): NodesOutcome => ({
  basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
  checks: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      reason: faker.helpers.arrayElement([
        "role_taken",
        "probe_fails",
        "probe_weak",
        "offline_unchecked",
        "reach_failed",
        "reached",
        "nothing_to_test",
        "node_is_controller",
        "node_holds_work",
        "nothing_held",
        "keeps_role",
      ] as const),
      ...(faker.datatype.boolean()
        ? {
            role: faker.helpers.arrayElement([
              "searching",
              "downloading",
              "storage",
              "converting",
              "subtitles",
              "metadata",
              "media-server-link",
              "notifying",
              "relaying",
              "serving",
            ] as const),
          }
        : {}),
      verdict: faker.helpers.arrayElement(["allow", "warn", "block"] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  effects: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getPlanRolesResponseNodesGainsMock() },
      { ...getPlanRolesResponseNodesLetsGoMock() },
      { ...getPlanRolesResponseNodesDrainsMock() },
      { ...getPlanRolesResponseNodesHandedBackMock() },
      { ...getPlanRolesResponseNodesPinnedMock() },
      { ...getPlanRolesResponseNodesMovesMock() },
      { ...getPlanRolesResponseNodesWritesOffMock() },
      { ...getPlanRolesResponseNodesLeavesMock() },
    ]),
  ),
  forfeit: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      nodeId: faker.string.uuid(),
      roles: faker.helpers.arrayElements([
        "searching",
        "downloading",
        "storage",
        "converting",
        "subtitles",
        "metadata",
        "media-server-link",
        "notifying",
        "relaying",
        "serving",
      ] as const),
    }),
  ),
  next: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    nodeId: faker.string.uuid(),
    roles: faker.helpers.arrayElements([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
  })),
  ...overrideResponse,
});

export const getGetNodeSettingsResponseMock = (
  overrideResponse: Partial<Extract<NodesSettings, object>> = {},
): NodesSettings => ({
  joinCodeLifetime: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  offlineGrace: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateNodeSettingsResponseMock = (
  overrideResponse: Partial<Extract<NodesSettings, object>> = {},
): NodesSettings => ({
  joinCodeLifetime: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  offlineGrace: {
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    ...(faker.datatype.boolean()
      ? { floor: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    min: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    step: faker.number.int(),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetNodeResponseMock = (
  overrideResponse: Partial<Extract<NodesNode, object>> = {},
): NodesNode => ({
  addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  dial: {
    ...(faker.datatype.boolean()
      ? { by: faker.helpers.arrayElement(["node", "controller"] as const) }
      : {}),
    controllerDials: Array.from(
      { length: faker.number.int({ min: 1, max: 10 }) },
      (_, i) => i + 1,
    ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
  },
  ...(faker.datatype.boolean()
    ? {
        facts: {
          appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
          assigned: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          httpAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
          restarting: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          running: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        },
      }
    : {}),
  id: faker.string.uuid(),
  intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  link: {
    ...(faker.datatype.boolean()
      ? {
          build: {
            arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
            installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            os: faker.string.alpha({ length: { min: 10, max: 20 } }),
            version: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
  },
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  probes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      fit: faker.helpers.arrayElement(["fits", "weak", "fails"] as const),
      probe: faker.helpers.arrayElement(["engine-starts", "data-writable", "listener-up"] as const),
      ...(faker.datatype.boolean()
        ? {
            reason: faker.helpers.arrayElement([
              "engine_not_listening",
              "downloads_not_writable",
              "downloads_on_network_share",
              "data_not_writable",
              "http_not_listening",
            ] as const),
          }
        : {}),
    }),
  ),
  roles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    ...(faker.datatype.boolean()
      ? {
          held: {
            bytes: faker.number.int(),
            noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
            units: faker.number.int(),
          },
        }
      : {}),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    state: faker.helpers.arrayElement([
      "holding",
      "away",
      "lapsed",
      "draining",
      "writing-off",
      "covering",
    ] as const),
  })),
  self: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getRenameNodeResponseMock = (
  overrideResponse: Partial<Extract<NodesNode, object>> = {},
): NodesNode => ({
  addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  dial: {
    ...(faker.datatype.boolean()
      ? { by: faker.helpers.arrayElement(["node", "controller"] as const) }
      : {}),
    controllerDials: Array.from(
      { length: faker.number.int({ min: 1, max: 10 }) },
      (_, i) => i + 1,
    ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
  },
  ...(faker.datatype.boolean()
    ? {
        facts: {
          appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
          assigned: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          httpAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
          restarting: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          running: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        },
      }
    : {}),
  id: faker.string.uuid(),
  intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  link: {
    ...(faker.datatype.boolean()
      ? {
          build: {
            arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
            installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            os: faker.string.alpha({ length: { min: 10, max: 20 } }),
            version: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
  },
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  probes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      fit: faker.helpers.arrayElement(["fits", "weak", "fails"] as const),
      probe: faker.helpers.arrayElement(["engine-starts", "data-writable", "listener-up"] as const),
      ...(faker.datatype.boolean()
        ? {
            reason: faker.helpers.arrayElement([
              "engine_not_listening",
              "downloads_not_writable",
              "downloads_on_network_share",
              "data_not_writable",
              "http_not_listening",
            ] as const),
          }
        : {}),
    }),
  ),
  roles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    ...(faker.datatype.boolean()
      ? {
          held: {
            bytes: faker.number.int(),
            noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
            units: faker.number.int(),
          },
        }
      : {}),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    state: faker.helpers.arrayElement([
      "holding",
      "away",
      "lapsed",
      "draining",
      "writing-off",
      "covering",
    ] as const),
  })),
  self: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getSetNodeDialResponseMock = (
  overrideResponse: Partial<Extract<NodesNode, object>> = {},
): NodesNode => ({
  addresses: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  dial: {
    ...(faker.datatype.boolean()
      ? { by: faker.helpers.arrayElement(["node", "controller"] as const) }
      : {}),
    controllerDials: Array.from(
      { length: faker.number.int({ min: 1, max: 10 }) },
      (_, i) => i + 1,
    ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
  },
  ...(faker.datatype.boolean()
    ? {
        facts: {
          appliedVersion: faker.string.alpha({ length: { min: 10, max: 20 } }),
          assigned: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          httpAddress: faker.string.alpha({ length: { min: 10, max: 20 } }),
          restarting: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          running: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        },
      }
    : {}),
  id: faker.string.uuid(),
  intentVersion: faker.helpers.fromRegExp("^[0-9a-f]{32}$"),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  link: {
    ...(faker.datatype.boolean()
      ? {
          build: {
            arch: faker.string.alpha({ length: { min: 10, max: 20 } }),
            installKind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            os: faker.string.alpha({ length: { min: 10, max: 20 } }),
            version: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
        }
      : {}),
    since: faker.date.past().toISOString().slice(0, 19) + "Z",
    state: faker.helpers.arrayElement(["local", "up", "down", "mismatched"] as const),
  },
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  pairedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  probes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      at: faker.date.past().toISOString().slice(0, 19) + "Z",
      fit: faker.helpers.arrayElement(["fits", "weak", "fails"] as const),
      probe: faker.helpers.arrayElement(["engine-starts", "data-writable", "listener-up"] as const),
      ...(faker.datatype.boolean()
        ? {
            reason: faker.helpers.arrayElement([
              "engine_not_listening",
              "downloads_not_writable",
              "downloads_on_network_share",
              "data_not_writable",
              "http_not_listening",
            ] as const),
          }
        : {}),
    }),
  ),
  roles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    ...(faker.datatype.boolean()
      ? {
          held: {
            bytes: faker.number.int(),
            noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
            units: faker.number.int(),
          },
        }
      : {}),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    state: faker.helpers.arrayElement([
      "holding",
      "away",
      "lapsed",
      "draining",
      "writing-off",
      "covering",
    ] as const),
  })),
  self: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getWriteOffResponseNodesGainsMock = (
  overrideResponse: Partial<NodesGains> = {},
): NodesGains => ({
  ...{
    kind: faker.helpers.arrayElement(["gains"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesLetsGoMock = (
  overrideResponse: Partial<NodesLetsGo> = {},
): NodesLetsGo => ({
  ...{
    kind: faker.helpers.arrayElement(["lets-go"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesDrainsMock = (
  overrideResponse: Partial<NodesDrains> = {},
): NodesDrains => ({
  ...{
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["drains"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesHandedBackMock = (
  overrideResponse: Partial<NodesHandedBack> = {},
): NodesHandedBack => ({
  ...{
    from: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    kind: faker.helpers.arrayElement(["handed-back"] as const),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    to: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesPinnedMock = (
  overrideResponse: Partial<NodesPinned> = {},
): NodesPinned => ({
  ...{
    kind: faker.helpers.arrayElement(["pinned"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesMovesMock = (
  overrideResponse: Partial<NodesMoves> = {},
): NodesMoves => ({
  ...{
    from: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["moves"] as const),
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    to: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesWritesOffMock = (
  overrideResponse: Partial<NodesWritesOff> = {},
): NodesWritesOff => ({
  ...{
    held: {
      bytes: faker.number.int(),
      noun: faker.string.alpha({ length: { min: 10, max: 20 } }),
      units: faker.number.int(),
    },
    kind: faker.helpers.arrayElement(["writes-off"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    role: faker.helpers.arrayElement([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseNodesLeavesMock = (
  overrideResponse: Partial<NodesLeaves> = {},
): NodesLeaves => ({
  ...{
    kind: faker.helpers.arrayElement(["leaves"] as const),
    node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getWriteOffResponseMock = (
  overrideResponse: Partial<Extract<NodesOutcome, object>> = {},
): NodesOutcome => ({
  basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
  checks: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      reason: faker.helpers.arrayElement([
        "role_taken",
        "probe_fails",
        "probe_weak",
        "offline_unchecked",
        "reach_failed",
        "reached",
        "nothing_to_test",
        "node_is_controller",
        "node_holds_work",
        "nothing_held",
        "keeps_role",
      ] as const),
      ...(faker.datatype.boolean()
        ? {
            role: faker.helpers.arrayElement([
              "searching",
              "downloading",
              "storage",
              "converting",
              "subtitles",
              "metadata",
              "media-server-link",
              "notifying",
              "relaying",
              "serving",
            ] as const),
          }
        : {}),
      verdict: faker.helpers.arrayElement(["allow", "warn", "block"] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  effects: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getWriteOffResponseNodesGainsMock() },
      { ...getWriteOffResponseNodesLetsGoMock() },
      { ...getWriteOffResponseNodesDrainsMock() },
      { ...getWriteOffResponseNodesHandedBackMock() },
      { ...getWriteOffResponseNodesPinnedMock() },
      { ...getWriteOffResponseNodesMovesMock() },
      { ...getWriteOffResponseNodesWritesOffMock() },
      { ...getWriteOffResponseNodesLeavesMock() },
    ]),
  ),
  forfeit: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      nodeId: faker.string.uuid(),
      roles: faker.helpers.arrayElements([
        "searching",
        "downloading",
        "storage",
        "converting",
        "subtitles",
        "metadata",
        "media-server-link",
        "notifying",
        "relaying",
        "serving",
      ] as const),
    }),
  ),
  next: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    nodeId: faker.string.uuid(),
    roles: faker.helpers.arrayElements([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ] as const),
  })),
  ...overrideResponse,
});

export const getListJoinCodesMockHandler = (
  overrideResponse?:
    | NodesOpenCodes
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesOpenCodes> | NodesOpenCodes),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/join-codes",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListJoinCodesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateJoinCodeMockHandler = (
  overrideResponse?:
    | NodesJoinCode
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NodesJoinCode> | NodesJoinCode),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/join-codes",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateJoinCodeResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getRevokeJoinCodeMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/join-codes/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetJoinCodeMockHandler = (
  overrideResponse?:
    | NodesCodeState
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesCodeState> | NodesCodeState),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/join-codes/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetJoinCodeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDialForCodeMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/join-codes/:id/dial",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 202 });
    },
    options,
  );
};

export const getListNodesMockHandler = (
  overrideResponse?:
    | NodesNodeList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesNodeList> | NodesNodeList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListNodesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetRoleBoardMockHandler = (
  overrideResponse?:
    | NodesRoleBoard
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesRoleBoard> | NodesRoleBoard),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/roles",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetRoleBoardResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetRolesMockHandler = (
  overrideResponse?:
    | NodesOutcome
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<NodesOutcome> | NodesOutcome),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/nodes/roles",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetRolesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPlanRolesMockHandler = (
  overrideResponse?:
    | NodesOutcome
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NodesOutcome> | NodesOutcome),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/nodes/roles/plan",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPlanRolesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetNodeSettingsMockHandler = (
  overrideResponse?:
    | NodesSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<NodesSettings> | NodesSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNodeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateNodeSettingsMockHandler = (
  overrideResponse?:
    | NodesSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<NodesSettings> | NodesSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/nodes/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateNodeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRemoveNodeMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/nodes/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetNodeMockHandler = (
  overrideResponse?:
    | NodesNode
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<NodesNode> | NodesNode),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNodeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRenameNodeMockHandler = (
  overrideResponse?:
    | NodesNode
    | ((info: Parameters<Parameters<typeof http.patch>[1]>[0]) => Promise<NodesNode> | NodesNode),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/nodes/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRenameNodeResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetNodeDialMockHandler = (
  overrideResponse?:
    | NodesNode
    | ((info: Parameters<Parameters<typeof http.put>[1]>[0]) => Promise<NodesNode> | NodesNode),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/nodes/:id/dial",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetNodeDialResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getWriteOffMockHandler = (
  overrideResponse?:
    | NodesOutcome
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<NodesOutcome> | NodesOutcome),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/nodes/:id/write-offs",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getWriteOffResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getNodesMock = () => [
  getListJoinCodesMockHandler(),
  getCreateJoinCodeMockHandler(),
  getRevokeJoinCodeMockHandler(),
  getGetJoinCodeMockHandler(),
  getDialForCodeMockHandler(),
  getListNodesMockHandler(),
  getGetRoleBoardMockHandler(),
  getSetRolesMockHandler(),
  getPlanRolesMockHandler(),
  getGetNodeSettingsMockHandler(),
  getUpdateNodeSettingsMockHandler(),
  getRemoveNodeMockHandler(),
  getGetNodeMockHandler(),
  getRenameNodeMockHandler(),
  getSetNodeDialMockHandler(),
  getWriteOffMockHandler(),
];
