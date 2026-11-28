import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  CanceledOutputBody,
  ConversionsConversionGroup,
  ConversionsConversionList,
  ConversionsConversionRow,
  ConversionsGroupPage,
  ConversionsNodeConverting,
} from "../../model";

export const getListConversionsResponseConversionsConversionRowMock = (
  overrideResponse: Partial<ConversionsConversionRow> = {},
): ConversionsConversionRow => ({
  ...{
    attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        endedAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        n: faker.number.int({ min: 1 }),
        node: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? {
              outcome: faker.helpers.arrayElement([
                "failed",
                "stalled",
                "disk_full",
                "put_back",
              ] as const),
            }
          : {}),
        startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      }),
    ),
    failure: faker.helpers.arrayElement([
      {
        class: faker.helpers.arrayElement(["run", "source"] as const),
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
        nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      },
      null,
    ]),
    finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    id: faker.string.uuid(),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    kind: faker.helpers.arrayElement(["conversion"] as const),
    lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
    node: faker.helpers.arrayElement([
      { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      null,
    ]),
    pausedBy: faker.helpers.arrayElement([
      Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
        faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      null,
    ]),
    progress: faker.helpers.arrayElement([
      {
        percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
        secondsLeft: faker.number.int({ min: 0 }),
        speed: faker.number.float({ min: 0, fractionDigits: 2 }),
      },
      null,
    ]),
    source: {
      edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
      fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      target: faker.string.alpha({ length: { min: 10, max: 20 } }),
      targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    },
    stage: faker.helpers.arrayElement([
      "queued",
      "copying",
      "ready",
      "running",
      "paused",
      "checking",
      "returning",
      "landing",
      "done",
      "failed",
      "canceled",
    ] as const),
    start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    target: {
      id: faker.helpers.arrayElement([faker.string.uuid(), null]),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    wait: faker.helpers.arrayElement([
      {
        ...(faker.datatype.boolean()
          ? {
              cause: faker.helpers.arrayElement([
                "no_role",
                "lapsed",
                "runs_elsewhere",
                "pinned_elsewhere",
                "failed_here",
                "untested",
                "test_failed",
                "retesting",
                "proof_lost",
                "processor_off",
                "none_online",
                "no_holder",
              ] as const),
            }
          : {}),
        code: faker.helpers.arrayElement([
          "in_line",
          "hours",
          "room",
          "copying",
          "away",
          "download",
          "testing",
          "transcode",
          "no_node",
          "no_source",
        ] as const),
        ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
        ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
        ...(faker.datatype.boolean()
          ? {
              node: {
                id: faker.string.uuid(),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
            }
          : {}),
        ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
        servers: faker.helpers.arrayElement([
          Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
            faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
          null,
        ]),
        ...(faker.datatype.boolean()
          ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
          : {}),
      },
      null,
    ]),
    work: {
      id: faker.string.uuid(),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
      type: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
  },
  ...overrideResponse,
});

export const getListConversionsResponseConversionsConversionGroupMock = (
  overrideResponse: Partial<ConversionsConversionGroup> = {},
): ConversionsConversionGroup => ({
  ...{
    bytes: faker.number.int({ min: 0 }),
    count: faker.number.int({ min: 1 }),
    finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    groupKind: faker.helpers.arrayElement(["backfill", "season"] as const),
    id: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["group"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    seconds: faker.number.int({ min: 0 }),
    start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  ...overrideResponse,
});

export const getListConversionsResponseMock = (
  overrideResponse: Partial<Extract<ConversionsConversionList, object>> = {},
): ConversionsConversionList => ({
  attention: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          code: faker.string.alpha({ length: { min: 10, max: 20 } }),
          endedAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          n: faker.number.int({ min: 1 }),
          node: faker.string.alpha({ length: { min: 10, max: 20 } }),
          ...(faker.datatype.boolean()
            ? {
                outcome: faker.helpers.arrayElement([
                  "failed",
                  "stalled",
                  "disk_full",
                  "put_back",
                ] as const),
              }
            : {}),
          startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        }),
      ),
      failure: faker.helpers.arrayElement([
        {
          class: faker.helpers.arrayElement(["run", "source"] as const),
          code: faker.string.alpha({ length: { min: 10, max: 20 } }),
          detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
          nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
        },
        null,
      ]),
      finish: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      id: faker.string.uuid(),
      items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      kind: faker.helpers.arrayElement(["conversion"] as const),
      lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
      node: faker.helpers.arrayElement([
        { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
        null,
      ]),
      pausedBy: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      progress: faker.helpers.arrayElement([
        {
          percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
          secondsLeft: faker.number.int({ min: 0 }),
          speed: faker.number.float({ min: 0, fractionDigits: 2 }),
        },
        null,
      ]),
      source: {
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        target: faker.string.alpha({ length: { min: 10, max: 20 } }),
        targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      },
      stage: faker.helpers.arrayElement([
        "queued",
        "copying",
        "ready",
        "running",
        "paused",
        "checking",
        "returning",
        "landing",
        "done",
        "failed",
        "canceled",
      ] as const),
      start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      target: {
        id: faker.helpers.arrayElement([faker.string.uuid(), null]),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      wait: faker.helpers.arrayElement([
        {
          ...(faker.datatype.boolean()
            ? {
                cause: faker.helpers.arrayElement([
                  "no_role",
                  "lapsed",
                  "runs_elsewhere",
                  "pinned_elsewhere",
                  "failed_here",
                  "untested",
                  "test_failed",
                  "retesting",
                  "proof_lost",
                  "processor_off",
                  "none_online",
                  "no_holder",
                ] as const),
              }
            : {}),
          code: faker.helpers.arrayElement([
            "in_line",
            "hours",
            "room",
            "copying",
            "away",
            "download",
            "testing",
            "transcode",
            "no_node",
            "no_source",
          ] as const),
          ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
          ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
          ...(faker.datatype.boolean()
            ? {
                node: {
                  id: faker.string.uuid(),
                  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
                },
              }
            : {}),
          ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
          servers: faker.helpers.arrayElement([
            Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
              faker.string.alpha({ length: { min: 10, max: 20 } }),
            ),
            null,
          ]),
          ...(faker.datatype.boolean()
            ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
          ...(faker.datatype.boolean()
            ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
            : {}),
        },
        null,
      ]),
      work: {
        id: faker.string.uuid(),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        type: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
    }),
  ),
  converting: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          code: faker.string.alpha({ length: { min: 10, max: 20 } }),
          endedAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          n: faker.number.int({ min: 1 }),
          node: faker.string.alpha({ length: { min: 10, max: 20 } }),
          ...(faker.datatype.boolean()
            ? {
                outcome: faker.helpers.arrayElement([
                  "failed",
                  "stalled",
                  "disk_full",
                  "put_back",
                ] as const),
              }
            : {}),
          startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        }),
      ),
      failure: faker.helpers.arrayElement([
        {
          class: faker.helpers.arrayElement(["run", "source"] as const),
          code: faker.string.alpha({ length: { min: 10, max: 20 } }),
          detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
          nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
        },
        null,
      ]),
      finish: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      id: faker.string.uuid(),
      items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      kind: faker.helpers.arrayElement(["conversion"] as const),
      lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
      node: faker.helpers.arrayElement([
        { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
        null,
      ]),
      pausedBy: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      progress: faker.helpers.arrayElement([
        {
          percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
          secondsLeft: faker.number.int({ min: 0 }),
          speed: faker.number.float({ min: 0, fractionDigits: 2 }),
        },
        null,
      ]),
      source: {
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        target: faker.string.alpha({ length: { min: 10, max: 20 } }),
        targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      },
      stage: faker.helpers.arrayElement([
        "queued",
        "copying",
        "ready",
        "running",
        "paused",
        "checking",
        "returning",
        "landing",
        "done",
        "failed",
        "canceled",
      ] as const),
      start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      target: {
        id: faker.helpers.arrayElement([faker.string.uuid(), null]),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      wait: faker.helpers.arrayElement([
        {
          ...(faker.datatype.boolean()
            ? {
                cause: faker.helpers.arrayElement([
                  "no_role",
                  "lapsed",
                  "runs_elsewhere",
                  "pinned_elsewhere",
                  "failed_here",
                  "untested",
                  "test_failed",
                  "retesting",
                  "proof_lost",
                  "processor_off",
                  "none_online",
                  "no_holder",
                ] as const),
              }
            : {}),
          code: faker.helpers.arrayElement([
            "in_line",
            "hours",
            "room",
            "copying",
            "away",
            "download",
            "testing",
            "transcode",
            "no_node",
            "no_source",
          ] as const),
          ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
          ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
          ...(faker.datatype.boolean()
            ? {
                node: {
                  id: faker.string.uuid(),
                  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
                },
              }
            : {}),
          ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
          servers: faker.helpers.arrayElement([
            Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
              faker.string.alpha({ length: { min: 10, max: 20 } }),
            ),
            null,
          ]),
          ...(faker.datatype.boolean()
            ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
            : {}),
          ...(faker.datatype.boolean()
            ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
            : {}),
        },
        null,
      ]),
      work: {
        id: faker.string.uuid(),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        type: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
    }),
  ),
  counts: {
    clearsAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    converting: faker.number.int({ min: 0 }),
    failed: faker.number.int({ min: 0 }),
    paused: faker.number.int({ min: 0 }),
    waiting: faker.number.int({ min: 0 }),
  },
  cursor: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  next: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
    rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getListConversionsResponseConversionsConversionRowMock() },
        { ...getListConversionsResponseConversionsConversionGroupMock() },
      ]),
    ),
  })),
  ...overrideResponse,
});

export const getGetConversionGroupResponseMock = (
  overrideResponse: Partial<Extract<ConversionsGroupPage, object>> = {},
): ConversionsGroupPage => ({
  cursor: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        endedAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        n: faker.number.int({ min: 1 }),
        node: faker.string.alpha({ length: { min: 10, max: 20 } }),
        ...(faker.datatype.boolean()
          ? {
              outcome: faker.helpers.arrayElement([
                "failed",
                "stalled",
                "disk_full",
                "put_back",
              ] as const),
            }
          : {}),
        startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      }),
    ),
    failure: faker.helpers.arrayElement([
      {
        class: faker.helpers.arrayElement(["run", "source"] as const),
        code: faker.string.alpha({ length: { min: 10, max: 20 } }),
        detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
        nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      },
      null,
    ]),
    finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    id: faker.string.uuid(),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    kind: faker.helpers.arrayElement(["conversion"] as const),
    lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
    node: faker.helpers.arrayElement([
      { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
      null,
    ]),
    pausedBy: faker.helpers.arrayElement([
      Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
        faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      null,
    ]),
    progress: faker.helpers.arrayElement([
      {
        percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
        secondsLeft: faker.number.int({ min: 0 }),
        speed: faker.number.float({ min: 0, fractionDigits: 2 }),
      },
      null,
    ]),
    source: {
      edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
      fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      target: faker.string.alpha({ length: { min: 10, max: 20 } }),
      targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    },
    stage: faker.helpers.arrayElement([
      "queued",
      "copying",
      "ready",
      "running",
      "paused",
      "checking",
      "returning",
      "landing",
      "done",
      "failed",
      "canceled",
    ] as const),
    start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    target: {
      id: faker.helpers.arrayElement([faker.string.uuid(), null]),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    wait: faker.helpers.arrayElement([
      {
        ...(faker.datatype.boolean()
          ? {
              cause: faker.helpers.arrayElement([
                "no_role",
                "lapsed",
                "runs_elsewhere",
                "pinned_elsewhere",
                "failed_here",
                "untested",
                "test_failed",
                "retesting",
                "proof_lost",
                "processor_off",
                "none_online",
                "no_holder",
              ] as const),
            }
          : {}),
        code: faker.helpers.arrayElement([
          "in_line",
          "hours",
          "room",
          "copying",
          "away",
          "download",
          "testing",
          "transcode",
          "no_node",
          "no_source",
        ] as const),
        ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
        ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
        ...(faker.datatype.boolean()
          ? {
              node: {
                id: faker.string.uuid(),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
            }
          : {}),
        ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
        servers: faker.helpers.arrayElement([
          Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
            faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
          null,
        ]),
        ...(faker.datatype.boolean()
          ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
          : {}),
      },
      null,
    ]),
    work: {
      id: faker.string.uuid(),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
      type: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
  })),
  ...overrideResponse,
});

export const getCancelConversionGroupResponseMock = (
  overrideResponse: Partial<Extract<CanceledOutputBody, object>> = {},
): CanceledOutputBody => ({ canceled: faker.number.int({ min: 0 }), ...overrideResponse });

export const getCancelConversionResponseMock = (
  overrideResponse: Partial<Extract<ConversionsConversionRow, object>> = {},
): ConversionsConversionRow => ({
  attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      endedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      n: faker.number.int({ min: 1 }),
      node: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? {
            outcome: faker.helpers.arrayElement([
              "failed",
              "stalled",
              "disk_full",
              "put_back",
            ] as const),
          }
        : {}),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  failure: faker.helpers.arrayElement([
    {
      class: faker.helpers.arrayElement(["run", "source"] as const),
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    },
    null,
  ]),
  finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.string.uuid(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  kind: faker.helpers.arrayElement(["conversion"] as const),
  lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
  node: faker.helpers.arrayElement([
    { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  pausedBy: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  progress: faker.helpers.arrayElement([
    {
      percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
      secondsLeft: faker.number.int({ min: 0 }),
      speed: faker.number.float({ min: 0, fractionDigits: 2 }),
    },
    null,
  ]),
  source: {
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    target: faker.string.alpha({ length: { min: 10, max: 20 } }),
    targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  },
  stage: faker.helpers.arrayElement([
    "queued",
    "copying",
    "ready",
    "running",
    "paused",
    "checking",
    "returning",
    "landing",
    "done",
    "failed",
    "canceled",
  ] as const),
  start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  target: {
    id: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  wait: faker.helpers.arrayElement([
    {
      ...(faker.datatype.boolean()
        ? {
            cause: faker.helpers.arrayElement([
              "no_role",
              "lapsed",
              "runs_elsewhere",
              "pinned_elsewhere",
              "failed_here",
              "untested",
              "test_failed",
              "retesting",
              "proof_lost",
              "processor_off",
              "none_online",
              "no_holder",
            ] as const),
          }
        : {}),
      code: faker.helpers.arrayElement([
        "in_line",
        "hours",
        "room",
        "copying",
        "away",
        "download",
        "testing",
        "transcode",
        "no_node",
        "no_source",
      ] as const),
      ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean()
        ? {
            node: {
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
      servers: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      ...(faker.datatype.boolean()
        ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
    },
    null,
  ]),
  work: {
    id: faker.string.uuid(),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    type: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getMoveConversionResponseMock = (
  overrideResponse: Partial<Extract<ConversionsConversionRow, object>> = {},
): ConversionsConversionRow => ({
  attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      endedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      n: faker.number.int({ min: 1 }),
      node: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? {
            outcome: faker.helpers.arrayElement([
              "failed",
              "stalled",
              "disk_full",
              "put_back",
            ] as const),
          }
        : {}),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  failure: faker.helpers.arrayElement([
    {
      class: faker.helpers.arrayElement(["run", "source"] as const),
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    },
    null,
  ]),
  finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.string.uuid(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  kind: faker.helpers.arrayElement(["conversion"] as const),
  lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
  node: faker.helpers.arrayElement([
    { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  pausedBy: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  progress: faker.helpers.arrayElement([
    {
      percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
      secondsLeft: faker.number.int({ min: 0 }),
      speed: faker.number.float({ min: 0, fractionDigits: 2 }),
    },
    null,
  ]),
  source: {
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    target: faker.string.alpha({ length: { min: 10, max: 20 } }),
    targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  },
  stage: faker.helpers.arrayElement([
    "queued",
    "copying",
    "ready",
    "running",
    "paused",
    "checking",
    "returning",
    "landing",
    "done",
    "failed",
    "canceled",
  ] as const),
  start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  target: {
    id: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  wait: faker.helpers.arrayElement([
    {
      ...(faker.datatype.boolean()
        ? {
            cause: faker.helpers.arrayElement([
              "no_role",
              "lapsed",
              "runs_elsewhere",
              "pinned_elsewhere",
              "failed_here",
              "untested",
              "test_failed",
              "retesting",
              "proof_lost",
              "processor_off",
              "none_online",
              "no_holder",
            ] as const),
          }
        : {}),
      code: faker.helpers.arrayElement([
        "in_line",
        "hours",
        "room",
        "copying",
        "away",
        "download",
        "testing",
        "transcode",
        "no_node",
        "no_source",
      ] as const),
      ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean()
        ? {
            node: {
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
      servers: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      ...(faker.datatype.boolean()
        ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
    },
    null,
  ]),
  work: {
    id: faker.string.uuid(),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    type: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getMoveConversionUpResponseMock = (
  overrideResponse: Partial<Extract<ConversionsConversionRow, object>> = {},
): ConversionsConversionRow => ({
  attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      endedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      n: faker.number.int({ min: 1 }),
      node: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? {
            outcome: faker.helpers.arrayElement([
              "failed",
              "stalled",
              "disk_full",
              "put_back",
            ] as const),
          }
        : {}),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  failure: faker.helpers.arrayElement([
    {
      class: faker.helpers.arrayElement(["run", "source"] as const),
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    },
    null,
  ]),
  finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.string.uuid(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  kind: faker.helpers.arrayElement(["conversion"] as const),
  lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
  node: faker.helpers.arrayElement([
    { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  pausedBy: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  progress: faker.helpers.arrayElement([
    {
      percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
      secondsLeft: faker.number.int({ min: 0 }),
      speed: faker.number.float({ min: 0, fractionDigits: 2 }),
    },
    null,
  ]),
  source: {
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    target: faker.string.alpha({ length: { min: 10, max: 20 } }),
    targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  },
  stage: faker.helpers.arrayElement([
    "queued",
    "copying",
    "ready",
    "running",
    "paused",
    "checking",
    "returning",
    "landing",
    "done",
    "failed",
    "canceled",
  ] as const),
  start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  target: {
    id: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  wait: faker.helpers.arrayElement([
    {
      ...(faker.datatype.boolean()
        ? {
            cause: faker.helpers.arrayElement([
              "no_role",
              "lapsed",
              "runs_elsewhere",
              "pinned_elsewhere",
              "failed_here",
              "untested",
              "test_failed",
              "retesting",
              "proof_lost",
              "processor_off",
              "none_online",
              "no_holder",
            ] as const),
          }
        : {}),
      code: faker.helpers.arrayElement([
        "in_line",
        "hours",
        "room",
        "copying",
        "away",
        "download",
        "testing",
        "transcode",
        "no_node",
        "no_source",
      ] as const),
      ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean()
        ? {
            node: {
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
      servers: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      ...(faker.datatype.boolean()
        ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
    },
    null,
  ]),
  work: {
    id: faker.string.uuid(),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    type: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getRetryConversionResponseMock = (
  overrideResponse: Partial<Extract<ConversionsConversionRow, object>> = {},
): ConversionsConversionRow => ({
  attempts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      endedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      n: faker.number.int({ min: 1 }),
      node: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? {
            outcome: faker.helpers.arrayElement([
              "failed",
              "stalled",
              "disk_full",
              "put_back",
            ] as const),
          }
        : {}),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  failure: faker.helpers.arrayElement([
    {
      class: faker.helpers.arrayElement(["run", "source"] as const),
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      detail: faker.string.alpha({ length: { min: 10, max: 20 } }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    },
    null,
  ]),
  finish: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    id: faker.string.uuid(),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  })),
  kind: faker.helpers.arrayElement(["conversion"] as const),
  lane: faker.helpers.arrayElement(["moved_up", "new", "upgrade", "backfill"] as const),
  node: faker.helpers.arrayElement([
    { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  pausedBy: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    null,
  ]),
  progress: faker.helpers.arrayElement([
    {
      percent: faker.number.float({ min: 0, max: 100, fractionDigits: 2 }),
      secondsLeft: faker.number.int({ min: 0 }),
      speed: faker.number.float({ min: 0, fractionDigits: 2 }),
    },
    null,
  ]),
  source: {
    edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    target: faker.string.alpha({ length: { min: 10, max: 20 } }),
    targetId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  },
  stage: faker.helpers.arrayElement([
    "queued",
    "copying",
    "ready",
    "running",
    "paused",
    "checking",
    "returning",
    "landing",
    "done",
    "failed",
    "canceled",
  ] as const),
  start: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  target: {
    id: faker.helpers.arrayElement([faker.string.uuid(), null]),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  wait: faker.helpers.arrayElement([
    {
      ...(faker.datatype.boolean()
        ? {
            cause: faker.helpers.arrayElement([
              "no_role",
              "lapsed",
              "runs_elsewhere",
              "pinned_elsewhere",
              "failed_here",
              "untested",
              "test_failed",
              "retesting",
              "proof_lost",
              "processor_off",
              "none_online",
              "no_holder",
            ] as const),
          }
        : {}),
      code: faker.helpers.arrayElement([
        "in_line",
        "hours",
        "room",
        "copying",
        "away",
        "download",
        "testing",
        "transcode",
        "no_node",
        "no_source",
      ] as const),
      ...(faker.datatype.boolean() ? { free: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean() ? { need: faker.number.int({ min: 0 }) } : {}),
      ...(faker.datatype.boolean()
        ? {
            node: {
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      ...(faker.datatype.boolean() ? { place: faker.number.int({ min: 1 }) } : {}),
      servers: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
        null,
      ]),
      ...(faker.datatype.boolean()
        ? { sourceTarget: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { until: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
    },
    null,
  ]),
  work: {
    id: faker.string.uuid(),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    type: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetNodeConvertingResponseMock = (
  overrideResponse: Partial<Extract<ConversionsNodeConverting, object>> = {},
): ConversionsNodeConverting => ({
  atOnce: faker.number.int({ min: 1 }),
  atOnceDefault: faker.number.int({ min: 1 }),
  hours: {
    on: faker.datatype.boolean(),
    window: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
  },
  override: faker.datatype.boolean(),
  processor: faker.datatype.boolean(),
  proofs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      api: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["qsv", "vaapi", "cuda", "videotoolbox"] as const),
        null,
      ]),
      class: faker.helpers.arrayElement([
        "h264_8",
        "hevc_8",
        "hevc_10",
        "av1_8",
        "av1_10",
        "hdr10",
        "hlg",
        "dv5",
        "dv81",
        "decode_sdr",
        "decode_hdr10",
      ] as const),
      device: faker.string.alpha({ length: { min: 10, max: 20 } }),
      error: faker.string.alpha({ length: { min: 10, max: 20 } }),
      failedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      lostAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      path: faker.helpers.arrayElement(["hardware", "hardware_opencl", "processor"] as const),
      pauses: faker.datatype.boolean(),
      provenAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      shape: {
        codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
        curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
        depth: faker.helpers.arrayElement([8, 10] as const),
        keepDolbyVision: faker.datatype.boolean(),
        range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
      },
      speed: faker.number.float({ min: 0, fractionDigits: 2 }),
    }),
  ),
  running: faker.number.int({ min: 0 }),
  scratch: faker.helpers.arrayElement([
    { free: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) },
    null,
  ]),
  sentence: {
    api: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["qsv", "vaapi", "cuda", "videotoolbox"] as const),
      null,
    ]),
    none: faker.datatype.boolean(),
    processor: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getUpdateNodeConvertingResponseMock = (
  overrideResponse: Partial<Extract<ConversionsNodeConverting, object>> = {},
): ConversionsNodeConverting => ({
  atOnce: faker.number.int({ min: 1 }),
  atOnceDefault: faker.number.int({ min: 1 }),
  hours: {
    on: faker.datatype.boolean(),
    window: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
  },
  override: faker.datatype.boolean(),
  processor: faker.datatype.boolean(),
  proofs: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      api: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["qsv", "vaapi", "cuda", "videotoolbox"] as const),
        null,
      ]),
      class: faker.helpers.arrayElement([
        "h264_8",
        "hevc_8",
        "hevc_10",
        "av1_8",
        "av1_10",
        "hdr10",
        "hlg",
        "dv5",
        "dv81",
        "decode_sdr",
        "decode_hdr10",
      ] as const),
      device: faker.string.alpha({ length: { min: 10, max: 20 } }),
      error: faker.string.alpha({ length: { min: 10, max: 20 } }),
      failedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      lostAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      path: faker.helpers.arrayElement(["hardware", "hardware_opencl", "processor"] as const),
      pauses: faker.datatype.boolean(),
      provenAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      shape: {
        codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
        curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
        depth: faker.helpers.arrayElement([8, 10] as const),
        keepDolbyVision: faker.datatype.boolean(),
        range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
      },
      speed: faker.number.float({ min: 0, fractionDigits: 2 }),
    }),
  ),
  running: faker.number.int({ min: 0 }),
  scratch: faker.helpers.arrayElement([
    { free: faker.number.int({ min: 0 }), total: faker.number.int({ min: 0 }) },
    null,
  ]),
  sentence: {
    api: faker.helpers.arrayElement([
      faker.helpers.arrayElement(["qsv", "vaapi", "cuda", "videotoolbox"] as const),
      null,
    ]),
    none: faker.datatype.boolean(),
    processor: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getListConversionsMockHandler = (
  overrideResponse?:
    | ConversionsConversionList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ConversionsConversionList> | ConversionsConversionList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/conversions",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListConversionsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetConversionGroupMockHandler = (
  overrideResponse?:
    | ConversionsGroupPage
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ConversionsGroupPage> | ConversionsGroupPage),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/conversions/groups/:groupId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetConversionGroupResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelConversionGroupMockHandler = (
  overrideResponse?:
    | CanceledOutputBody
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<CanceledOutputBody> | CanceledOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/conversions/groups/:groupId/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelConversionGroupResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCancelConversionMockHandler = (
  overrideResponse?:
    | ConversionsConversionRow
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ConversionsConversionRow> | ConversionsConversionRow),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/conversions/:id/cancel",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCancelConversionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getMoveConversionMockHandler = (
  overrideResponse?:
    | ConversionsConversionRow
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ConversionsConversionRow> | ConversionsConversionRow),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/conversions/:id/move",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getMoveConversionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getMoveConversionUpMockHandler = (
  overrideResponse?:
    | ConversionsConversionRow
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ConversionsConversionRow> | ConversionsConversionRow),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/conversions/:id/move-up",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getMoveConversionUpResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRetryConversionMockHandler = (
  overrideResponse?:
    | ConversionsConversionRow
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ConversionsConversionRow> | ConversionsConversionRow),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/conversions/:id/retry",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRetryConversionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetNodeConvertingMockHandler = (
  overrideResponse?:
    | ConversionsNodeConverting
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ConversionsNodeConverting> | ConversionsNodeConverting),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/nodes/:id/converting",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetNodeConvertingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateNodeConvertingMockHandler = (
  overrideResponse?:
    | ConversionsNodeConverting
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<ConversionsNodeConverting> | ConversionsNodeConverting),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/nodes/:id/converting",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateNodeConvertingResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRerunNodeTestsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/nodes/:id/converting/rerun",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 202 });
    },
    options,
  );
};
export const getConversionsMock = () => [
  getListConversionsMockHandler(),
  getGetConversionGroupMockHandler(),
  getCancelConversionGroupMockHandler(),
  getCancelConversionMockHandler(),
  getMoveConversionMockHandler(),
  getMoveConversionUpMockHandler(),
  getRetryConversionMockHandler(),
  getGetNodeConvertingMockHandler(),
  getUpdateNodeConvertingMockHandler(),
  getRerunNodeTestsMockHandler(),
];
