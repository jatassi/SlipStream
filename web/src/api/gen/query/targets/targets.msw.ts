import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  TargetsAnyCut,
  TargetsCut,
  TargetsCutChange,
  TargetsCutList,
  TargetsNamedCuts,
  TargetsProfile,
  TargetsProfileChange,
  TargetsProfileDetail,
  TargetsProfileList,
  TargetsQualityList,
  TargetsTarget,
  TargetsTargetChange,
  TargetsTargetList,
  TargetsWants,
  TargetsWantsChange,
} from "../../model";

export const getListCutsResponseMock = (
  overrideResponse: Partial<Extract<TargetsCutList, object>> = {},
): TargetsCutList => ({
  cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    aliases: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$") }
      : {}),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    origin: faker.helpers.arrayElement(["builtin", "type", "custom"] as const),
  })),
  ...overrideResponse,
});

export const getCreateCutResponseMock = (
  overrideResponse: Partial<Extract<TargetsCut, object>> = {},
): TargetsCut => ({
  aliases: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...(faker.datatype.boolean()
    ? { contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$") }
    : {}),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  origin: faker.helpers.arrayElement(["builtin", "type", "custom"] as const),
  ...overrideResponse,
});

export const getUpdateCutResponseMock = (
  overrideResponse: Partial<Extract<TargetsCutChange, object>> = {},
): TargetsCutChange => ({
  cut: {
    aliases: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$") }
      : {}),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    origin: faker.helpers.arrayElement(["builtin", "type", "custom"] as const),
  },
  impact: {
    downloading: faker.number.int({ min: 0 }),
    made: {
      bytes: faker.number.int({ min: 0 }),
      files: faker.number.int({ min: 0 }),
      matching: faker.number.int({ min: 0 }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
      seconds: faker.number.int({ min: 0 }),
      unconvertible: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.uuid()),
      works: faker.number.int({ min: 0 }),
    },
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    remake: {
      older: faker.number.int({ min: 0 }),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
    },
    renamed: faker.number.int({ min: 0 }),
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
    subtitles: {
      days: faker.number.int({ min: 0 }),
      dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          deleted: faker.number.int({ min: 0 }),
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      gained: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needScan: faker.number.int({ min: 0 }),
          searches: faker.number.int({ min: 0 }),
        }),
      ),
    },
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getListProfilesResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getListProfilesResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getListProfilesResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfileList, object>> = {},
): TargetsProfileList => ({
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      default: faker.datatype.boolean(),
      editions: faker.helpers.arrayElement([
        { ...getListProfilesResponseTargetsAnyCutMock() },
        { ...getListProfilesResponseTargetsNamedCutsMock() },
      ]),
      id: faker.string.uuid(),
      languages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          audio: faker.helpers.arrayElement(["always", "no_audio"] as const),
          forced: faker.datatype.boolean(),
          hearingImpaired: faker.helpers.arrayElement(["avoid", "allow", "prefer"] as const),
          language: faker.string.alpha({ length: { min: 2, max: 5 } }),
        }),
      ),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean() ? { fromTargetId: faker.string.uuid() } : {}),
          ...(faker.datatype.boolean()
            ? { graceSeconds: faker.number.int({ min: 3600, max: 7776000 }) }
            : {}),
          sourcing: faker.helpers.arrayElement([
            "download",
            "derive",
            "download_then_derive",
          ] as const),
          targetId: faker.string.uuid(),
        }),
      ),
      works: faker.number.int({ min: 0 }),
    }),
  ),
  ...overrideResponse,
});

export const getCreateProfileResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getCreateProfileResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getCreateProfileResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfile, object>> = {},
): TargetsProfile => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  default: faker.datatype.boolean(),
  editions: faker.helpers.arrayElement([
    { ...getCreateProfileResponseTargetsAnyCutMock() },
    { ...getCreateProfileResponseTargetsNamedCutsMock() },
  ]),
  id: faker.string.uuid(),
  languages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      audio: faker.helpers.arrayElement(["always", "no_audio"] as const),
      forced: faker.datatype.boolean(),
      hearingImpaired: faker.helpers.arrayElement(["avoid", "allow", "prefer"] as const),
      language: faker.string.alpha({ length: { min: 2, max: 5 } }),
    }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { fromTargetId: faker.string.uuid() } : {}),
      ...(faker.datatype.boolean()
        ? { graceSeconds: faker.number.int({ min: 3600, max: 7776000 }) }
        : {}),
      sourcing: faker.helpers.arrayElement(["download", "derive", "download_then_derive"] as const),
      targetId: faker.string.uuid(),
    }),
  ),
  works: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getDeleteProfileResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getDeleteProfileResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getDeleteProfileResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfileChange, object>> = {},
): TargetsProfileChange => ({
  impact: {
    downloading: faker.number.int({ min: 0 }),
    made: {
      bytes: faker.number.int({ min: 0 }),
      files: faker.number.int({ min: 0 }),
      matching: faker.number.int({ min: 0 }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
      seconds: faker.number.int({ min: 0 }),
      unconvertible: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.uuid()),
      works: faker.number.int({ min: 0 }),
    },
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    remake: {
      older: faker.number.int({ min: 0 }),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
    },
    renamed: faker.number.int({ min: 0 }),
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
    subtitles: {
      days: faker.number.int({ min: 0 }),
      dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          deleted: faker.number.int({ min: 0 }),
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      gained: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needScan: faker.number.int({ min: 0 }),
          searches: faker.number.int({ min: 0 }),
        }),
      ),
    },
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  profile: {
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    default: faker.datatype.boolean(),
    editions: faker.helpers.arrayElement([
      { ...getDeleteProfileResponseTargetsAnyCutMock() },
      { ...getDeleteProfileResponseTargetsNamedCutsMock() },
    ]),
    id: faker.string.uuid(),
    languages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        audio: faker.helpers.arrayElement(["always", "no_audio"] as const),
        forced: faker.datatype.boolean(),
        hearingImpaired: faker.helpers.arrayElement(["avoid", "allow", "prefer"] as const),
        language: faker.string.alpha({ length: { min: 2, max: 5 } }),
      }),
    ),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        ...(faker.datatype.boolean() ? { fromTargetId: faker.string.uuid() } : {}),
        ...(faker.datatype.boolean()
          ? { graceSeconds: faker.number.int({ min: 3600, max: 7776000 }) }
          : {}),
        sourcing: faker.helpers.arrayElement([
          "download",
          "derive",
          "download_then_derive",
        ] as const),
        targetId: faker.string.uuid(),
      }),
    ),
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getGetProfileResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getGetProfileResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getGetProfileResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfileDetail, object>> = {},
): TargetsProfileDetail => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  default: faker.datatype.boolean(),
  editions: faker.helpers.arrayElement([
    { ...getGetProfileResponseTargetsAnyCutMock() },
    { ...getGetProfileResponseTargetsNamedCutsMock() },
  ]),
  estimates: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      bytes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      mibPerMinute: faker.helpers.arrayElement([
        faker.number.float({ min: 0, fractionDigits: 2 }),
        null,
      ]),
      targetId: faker.string.uuid(),
    }),
  ),
  id: faker.string.uuid(),
  languages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      audio: faker.helpers.arrayElement(["always", "no_audio"] as const),
      forced: faker.datatype.boolean(),
      hearingImpaired: faker.helpers.arrayElement(["avoid", "allow", "prefer"] as const),
      language: faker.string.alpha({ length: { min: 2, max: 5 } }),
    }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { fromTargetId: faker.string.uuid() } : {}),
      ...(faker.datatype.boolean()
        ? { graceSeconds: faker.number.int({ min: 3600, max: 7776000 }) }
        : {}),
      sourcing: faker.helpers.arrayElement(["download", "derive", "download_then_derive"] as const),
      targetId: faker.string.uuid(),
    }),
  ),
  works: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getUpdateProfileResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getUpdateProfileResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getUpdateProfileResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfileChange, object>> = {},
): TargetsProfileChange => ({
  impact: {
    downloading: faker.number.int({ min: 0 }),
    made: {
      bytes: faker.number.int({ min: 0 }),
      files: faker.number.int({ min: 0 }),
      matching: faker.number.int({ min: 0 }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
      seconds: faker.number.int({ min: 0 }),
      unconvertible: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.uuid()),
      works: faker.number.int({ min: 0 }),
    },
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    remake: {
      older: faker.number.int({ min: 0 }),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
    },
    renamed: faker.number.int({ min: 0 }),
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
    subtitles: {
      days: faker.number.int({ min: 0 }),
      dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          deleted: faker.number.int({ min: 0 }),
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      gained: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needScan: faker.number.int({ min: 0 }),
          searches: faker.number.int({ min: 0 }),
        }),
      ),
    },
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  profile: {
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    default: faker.datatype.boolean(),
    editions: faker.helpers.arrayElement([
      { ...getUpdateProfileResponseTargetsAnyCutMock() },
      { ...getUpdateProfileResponseTargetsNamedCutsMock() },
    ]),
    id: faker.string.uuid(),
    languages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        audio: faker.helpers.arrayElement(["always", "no_audio"] as const),
        forced: faker.datatype.boolean(),
        hearingImpaired: faker.helpers.arrayElement(["avoid", "allow", "prefer"] as const),
        language: faker.string.alpha({ length: { min: 2, max: 5 } }),
      }),
    ),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        ...(faker.datatype.boolean() ? { fromTargetId: faker.string.uuid() } : {}),
        ...(faker.datatype.boolean()
          ? { graceSeconds: faker.number.int({ min: 3600, max: 7776000 }) }
          : {}),
        sourcing: faker.helpers.arrayElement([
          "download",
          "derive",
          "download_then_derive",
        ] as const),
        targetId: faker.string.uuid(),
      }),
    ),
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getMakeDefaultProfileResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getMakeDefaultProfileResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getMakeDefaultProfileResponseMock = (
  overrideResponse: Partial<Extract<TargetsProfile, object>> = {},
): TargetsProfile => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  default: faker.datatype.boolean(),
  editions: faker.helpers.arrayElement([
    { ...getMakeDefaultProfileResponseTargetsAnyCutMock() },
    { ...getMakeDefaultProfileResponseTargetsNamedCutsMock() },
  ]),
  id: faker.string.uuid(),
  languages: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      audio: faker.helpers.arrayElement(["always", "no_audio"] as const),
      forced: faker.datatype.boolean(),
      hearingImpaired: faker.helpers.arrayElement(["avoid", "allow", "prefer"] as const),
      language: faker.string.alpha({ length: { min: 2, max: 5 } }),
    }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { fromTargetId: faker.string.uuid() } : {}),
      ...(faker.datatype.boolean()
        ? { graceSeconds: faker.number.int({ min: 3600, max: 7776000 }) }
        : {}),
      sourcing: faker.helpers.arrayElement(["download", "derive", "download_then_derive"] as const),
      targetId: faker.string.uuid(),
    }),
  ),
  works: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getListQualitiesResponseMock = (
  overrideResponse: Partial<Extract<TargetsQualityList, object>> = {},
): TargetsQualityList => ({
  presets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      key: faker.helpers.arrayElement([
        "disc-or-resolution",
        "anything-better",
        "resolution-only",
      ] as const),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
            faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
      ),
    }),
  ),
  qualities: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      cinema: faker.datatype.boolean(),
      floor: faker.number.float({ min: 0, fractionDigits: 2 }),
      key: faker.helpers.arrayElement([
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
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      resolution: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["2160p", "1080p", "720p", "480p"] as const),
        null,
      ]),
      source: faker.helpers.arrayElement([
        "tv",
        "hdtv",
        "dvd",
        "webrip",
        "webdl",
        "bluray",
        "remux",
        "cinema",
      ] as const),
    }),
  ),
  ...overrideResponse,
});

export const getListTargetsResponseMock = (
  overrideResponse: Partial<Extract<TargetsTargetList, object>> = {},
): TargetsTargetList => ({
  targets: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.uuid(),
      ),
      conversion: faker.helpers.arrayElement([
        {
          audio: faker.helpers.arrayElement([
            "copy",
            "stereo_aac",
            "ac3_51_and_stereo_aac",
          ] as const),
          causes: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
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
            count: faker.number.int({ min: 1 }),
            node: faker.helpers.arrayElement([
              {
                id: faker.string.uuid(),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            notifiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            shape: faker.string.alpha({ length: { min: 10, max: 20 } }),
            why: faker.helpers.arrayElement([
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
          })),
          codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
          container: faker.helpers.arrayElement(["mkv", "mp4"] as const),
          curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
          defaultKbps: faker.number.int({ min: 100 }),
          depth: faker.helpers.arrayElement([8, 10] as const),
          dropCommentary: faker.datatype.boolean(),
          kbps: faker.number.int({ min: 100, max: 200000 }),
          keepDolbyVision: faker.datatype.boolean(),
          maxHeight: faker.helpers.arrayElement([2160, 1080, 720, 480] as const),
          older: faker.number.int({ min: 0 }),
          range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
          runOnNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          version: faker.number.int({ min: 1 }),
        },
        null,
      ]),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
      defaultKbps: faker.number.int({ min: 100 }),
      filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          attribute: faker.helpers.arrayElement([
            "hdr",
            "video_codec",
            "audio_codec",
            "channels",
            "language",
          ] as const),
          mode: faker.helpers.arrayElement(["only", "not"] as const),
          values: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
        }),
      ),
      holdSeconds: faker.number.int({ min: 0, max: 604800 }),
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      propers: faker.helpers.arrayElement([
        "prefer_and_upgrade",
        "prefer_no_upgrade",
        "dont_prefer",
      ] as const),
      scores: {
        formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({ formatId: faker.string.uuid(), score: faker.number.int() }),
        ),
        increment: faker.number.int({ min: 1 }),
        minimum: faker.number.int(),
        upgradeUntil: faker.number.int(),
      },
      sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          max: faker.helpers.arrayElement([
            faker.number.float({ min: 0, fractionDigits: 2 }),
            null,
          ]),
          min: faker.number.float({ min: 0, fractionDigits: 2 }),
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
        }),
      ),
      tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
            faker.string.alpha({ length: { min: 10, max: 20 } }),
          ),
      ),
    }),
  ),
  ...overrideResponse,
});

export const getCreateTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  conversion: faker.helpers.arrayElement([
    {
      audio: faker.helpers.arrayElement(["copy", "stereo_aac", "ac3_51_and_stereo_aac"] as const),
      causes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
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
          count: faker.number.int({ min: 1 }),
          node: faker.helpers.arrayElement([
            { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
            null,
          ]),
          notifiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          shape: faker.string.alpha({ length: { min: 10, max: 20 } }),
          why: faker.helpers.arrayElement([
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
        }),
      ),
      codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
      container: faker.helpers.arrayElement(["mkv", "mp4"] as const),
      curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
      defaultKbps: faker.number.int({ min: 100 }),
      depth: faker.helpers.arrayElement([8, 10] as const),
      dropCommentary: faker.datatype.boolean(),
      kbps: faker.number.int({ min: 100, max: 200000 }),
      keepDolbyVision: faker.datatype.boolean(),
      maxHeight: faker.helpers.arrayElement([2160, 1080, 720, 480] as const),
      older: faker.number.int({ min: 0 }),
      range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
      runOnNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      version: faker.number.int({ min: 1 }),
    },
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
  defaultKbps: faker.number.int({ min: 100 }),
  filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attribute: faker.helpers.arrayElement([
        "hdr",
        "video_codec",
        "audio_codec",
        "channels",
        "language",
      ] as const),
      mode: faker.helpers.arrayElement(["only", "not"] as const),
      values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  holdSeconds: faker.number.int({ min: 0, max: 604800 }),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  propers: faker.helpers.arrayElement([
    "prefer_and_upgrade",
    "prefer_no_upgrade",
    "dont_prefer",
  ] as const),
  scores: {
    formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({ formatId: faker.string.uuid(), score: faker.number.int() }),
    ),
    increment: faker.number.int({ min: 1 }),
    minimum: faker.number.int(),
    upgradeUntil: faker.number.int(),
  },
  sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
    min: faker.number.float({ min: 0, fractionDigits: 2 }),
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
  })),
  tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  ),
  ...overrideResponse,
});

export const getGetTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  conversion: faker.helpers.arrayElement([
    {
      audio: faker.helpers.arrayElement(["copy", "stereo_aac", "ac3_51_and_stereo_aac"] as const),
      causes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
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
          count: faker.number.int({ min: 1 }),
          node: faker.helpers.arrayElement([
            { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
            null,
          ]),
          notifiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          shape: faker.string.alpha({ length: { min: 10, max: 20 } }),
          why: faker.helpers.arrayElement([
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
        }),
      ),
      codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
      container: faker.helpers.arrayElement(["mkv", "mp4"] as const),
      curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
      defaultKbps: faker.number.int({ min: 100 }),
      depth: faker.helpers.arrayElement([8, 10] as const),
      dropCommentary: faker.datatype.boolean(),
      kbps: faker.number.int({ min: 100, max: 200000 }),
      keepDolbyVision: faker.datatype.boolean(),
      maxHeight: faker.helpers.arrayElement([2160, 1080, 720, 480] as const),
      older: faker.number.int({ min: 0 }),
      range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
      runOnNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      version: faker.number.int({ min: 1 }),
    },
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
  defaultKbps: faker.number.int({ min: 100 }),
  filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attribute: faker.helpers.arrayElement([
        "hdr",
        "video_codec",
        "audio_codec",
        "channels",
        "language",
      ] as const),
      mode: faker.helpers.arrayElement(["only", "not"] as const),
      values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  holdSeconds: faker.number.int({ min: 0, max: 604800 }),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  propers: faker.helpers.arrayElement([
    "prefer_and_upgrade",
    "prefer_no_upgrade",
    "dont_prefer",
  ] as const),
  scores: {
    formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({ formatId: faker.string.uuid(), score: faker.number.int() }),
    ),
    increment: faker.number.int({ min: 1 }),
    minimum: faker.number.int(),
    upgradeUntil: faker.number.int(),
  },
  sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
    min: faker.number.float({ min: 0, fractionDigits: 2 }),
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
  })),
  tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  ),
  ...overrideResponse,
});

export const getUpdateTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTargetChange, object>> = {},
): TargetsTargetChange => ({
  impact: {
    downloading: faker.number.int({ min: 0 }),
    made: {
      bytes: faker.number.int({ min: 0 }),
      files: faker.number.int({ min: 0 }),
      matching: faker.number.int({ min: 0 }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
      seconds: faker.number.int({ min: 0 }),
      unconvertible: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.uuid()),
      works: faker.number.int({ min: 0 }),
    },
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    remake: {
      older: faker.number.int({ min: 0 }),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
    },
    renamed: faker.number.int({ min: 0 }),
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
    subtitles: {
      days: faker.number.int({ min: 0 }),
      dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          deleted: faker.number.int({ min: 0 }),
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      gained: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needScan: faker.number.int({ min: 0 }),
          searches: faker.number.int({ min: 0 }),
        }),
      ),
    },
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  target: {
    contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.uuid(),
    ),
    conversion: faker.helpers.arrayElement([
      {
        audio: faker.helpers.arrayElement(["copy", "stereo_aac", "ac3_51_and_stereo_aac"] as const),
        causes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
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
            count: faker.number.int({ min: 1 }),
            node: faker.helpers.arrayElement([
              {
                id: faker.string.uuid(),
                name: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
              null,
            ]),
            notifiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            shape: faker.string.alpha({ length: { min: 10, max: 20 } }),
            why: faker.helpers.arrayElement([
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
          }),
        ),
        codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
        container: faker.helpers.arrayElement(["mkv", "mp4"] as const),
        curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
        defaultKbps: faker.number.int({ min: 100 }),
        depth: faker.helpers.arrayElement([8, 10] as const),
        dropCommentary: faker.datatype.boolean(),
        kbps: faker.number.int({ min: 100, max: 200000 }),
        keepDolbyVision: faker.datatype.boolean(),
        maxHeight: faker.helpers.arrayElement([2160, 1080, 720, 480] as const),
        older: faker.number.int({ min: 0 }),
        range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
        runOnNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        version: faker.number.int({ min: 1 }),
      },
      null,
    ]),
    createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
    defaultKbps: faker.number.int({ min: 100 }),
    filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        attribute: faker.helpers.arrayElement([
          "hdr",
          "video_codec",
          "audio_codec",
          "channels",
          "language",
        ] as const),
        mode: faker.helpers.arrayElement(["only", "not"] as const),
        values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      }),
    ),
    holdSeconds: faker.number.int({ min: 0, max: 604800 }),
    id: faker.string.uuid(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    propers: faker.helpers.arrayElement([
      "prefer_and_upgrade",
      "prefer_no_upgrade",
      "dont_prefer",
    ] as const),
    scores: {
      formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({ formatId: faker.string.uuid(), score: faker.number.int() }),
      ),
      increment: faker.number.int({ min: 1 }),
      minimum: faker.number.int(),
      upgradeUntil: faker.number.int(),
    },
    sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
        min: faker.number.float({ min: 0, fractionDigits: 2 }),
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
      }),
    ),
    tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
        faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    ),
  },
  ...overrideResponse,
});

export const getDuplicateTargetResponseMock = (
  overrideResponse: Partial<Extract<TargetsTarget, object>> = {},
): TargetsTarget => ({
  contains: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.uuid(),
  ),
  conversion: faker.helpers.arrayElement([
    {
      audio: faker.helpers.arrayElement(["copy", "stereo_aac", "ac3_51_and_stereo_aac"] as const),
      causes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
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
          count: faker.number.int({ min: 1 }),
          node: faker.helpers.arrayElement([
            { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
            null,
          ]),
          notifiedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          shape: faker.string.alpha({ length: { min: 10, max: 20 } }),
          why: faker.helpers.arrayElement([
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
        }),
      ),
      codec: faker.helpers.arrayElement(["h264", "hevc", "av1"] as const),
      container: faker.helpers.arrayElement(["mkv", "mp4"] as const),
      curve: faker.helpers.arrayElement(["bt2390", "hable", "mobius", "reinhard"] as const),
      defaultKbps: faker.number.int({ min: 100 }),
      depth: faker.helpers.arrayElement([8, 10] as const),
      dropCommentary: faker.datatype.boolean(),
      kbps: faker.number.int({ min: 100, max: 200000 }),
      keepDolbyVision: faker.datatype.boolean(),
      maxHeight: faker.helpers.arrayElement([2160, 1080, 720, 480] as const),
      older: faker.number.int({ min: 0 }),
      range: faker.helpers.arrayElement(["sdr", "hdr10", "source"] as const),
      runOnNodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      version: faker.number.int({ min: 1 }),
    },
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  cutoff: faker.string.alpha({ length: { min: 10, max: 20 } }),
  defaultKbps: faker.number.int({ min: 100 }),
  filters: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attribute: faker.helpers.arrayElement([
        "hdr",
        "video_codec",
        "audio_codec",
        "channels",
        "language",
      ] as const),
      mode: faker.helpers.arrayElement(["only", "not"] as const),
      values: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  holdSeconds: faker.number.int({ min: 0, max: 604800 }),
  id: faker.string.uuid(),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  profiles: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({ id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) }),
  ),
  propers: faker.helpers.arrayElement([
    "prefer_and_upgrade",
    "prefer_no_upgrade",
    "dont_prefer",
  ] as const),
  scores: {
    formats: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({ formatId: faker.string.uuid(), score: faker.number.int() }),
    ),
    increment: faker.number.int({ min: 1 }),
    minimum: faker.number.int(),
    upgradeUntil: faker.number.int(),
  },
  sizes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    max: faker.helpers.arrayElement([faker.number.float({ min: 0, fractionDigits: 2 }), null]),
    min: faker.number.float({ min: 0, fractionDigits: 2 }),
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
  })),
  tiers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  ),
  ...overrideResponse,
});

export const getGetWorkWantsResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getGetWorkWantsResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkWantsResponseMock = (
  overrideResponse: Partial<Extract<TargetsWants, object>> = {},
): TargetsWants => ({
  editions: faker.helpers.arrayElement([
    { ...getGetWorkWantsResponseTargetsAnyCutMock() },
    { ...getGetWorkWantsResponseTargetsNamedCutsMock() },
  ]),
  from: faker.helpers.arrayElement(["profile", "work"] as const),
  profileId: faker.string.uuid(),
  profileName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getSetWorkWantsResponseTargetsAnyCutMock = (
  overrideResponse: Partial<TargetsAnyCut> = {},
): TargetsAnyCut => ({
  ...{ kind: faker.helpers.arrayElement(["any"] as const) },
  ...overrideResponse,
});

export const getSetWorkWantsResponseTargetsNamedCutsMock = (
  overrideResponse: Partial<TargetsNamedCuts> = {},
): TargetsNamedCuts => ({
  ...{
    cuts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    kind: faker.helpers.arrayElement(["cuts"] as const),
  },
  ...overrideResponse,
});

export const getSetWorkWantsResponseMock = (
  overrideResponse: Partial<Extract<TargetsWantsChange, object>> = {},
): TargetsWantsChange => ({
  impact: {
    downloading: faker.number.int({ min: 0 }),
    made: {
      bytes: faker.number.int({ min: 0 }),
      files: faker.number.int({ min: 0 }),
      matching: faker.number.int({ min: 0 }),
      nodes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
      seconds: faker.number.int({ min: 0 }),
      unconvertible: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.string.uuid()),
      works: faker.number.int({ min: 0 }),
    },
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
    remake: {
      older: faker.number.int({ min: 0 }),
      rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          bytes: faker.number.int({ min: 0 }),
          files: faker.number.int({ min: 0 }),
          seconds: faker.number.int({ min: 0 }),
          title: faker.string.alpha({ length: { min: 10, max: 20 } }),
          workId: faker.string.uuid(),
        }),
      ),
    },
    renamed: faker.number.int({ min: 0 }),
    renames: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fileId: faker.string.uuid(),
            from: faker.string.alpha({ length: { min: 10, max: 20 } }),
            to: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        workId: faker.string.uuid(),
      }),
    ),
    subtitles: {
      days: faker.number.int({ min: 0 }),
      dropped: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          deleted: faker.number.int({ min: 0 }),
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      gained: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          forced: faker.datatype.boolean(),
          language: faker.string.alpha({ length: { min: 10, max: 20 } }),
          needScan: faker.number.int({ min: 0 }),
          searches: faker.number.int({ min: 0 }),
        }),
      ),
    },
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  wants: {
    editions: faker.helpers.arrayElement([
      { ...getSetWorkWantsResponseTargetsAnyCutMock() },
      { ...getSetWorkWantsResponseTargetsNamedCutsMock() },
    ]),
    from: faker.helpers.arrayElement(["profile", "work"] as const),
    profileId: faker.string.uuid(),
    profileName: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListCutsMockHandler = (
  overrideResponse?:
    | TargetsCutList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsCutList> | TargetsCutList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/cuts",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListCutsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateCutMockHandler = (
  overrideResponse?:
    | TargetsCut
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<TargetsCut> | TargetsCut),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/cuts",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateCutResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getDeleteCutMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/cuts/:cutKey",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getUpdateCutMockHandler = (
  overrideResponse?:
    | TargetsCutChange
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TargetsCutChange> | TargetsCutChange),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/cuts/:cutKey",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateCutResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListProfilesMockHandler = (
  overrideResponse?:
    | TargetsProfileList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsProfileList> | TargetsProfileList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/profiles",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListProfilesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateProfileMockHandler = (
  overrideResponse?:
    | TargetsProfile
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsProfile> | TargetsProfile),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/profiles",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateProfileResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getDeleteProfileMockHandler = (
  overrideResponse?:
    | TargetsProfileChange
    | ((
        info: Parameters<Parameters<typeof http.delete>[1]>[0],
      ) => Promise<TargetsProfileChange> | TargetsProfileChange),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/profiles/:profileId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getDeleteProfileResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetProfileMockHandler = (
  overrideResponse?:
    | TargetsProfileDetail
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsProfileDetail> | TargetsProfileDetail),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/profiles/:profileId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetProfileResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateProfileMockHandler = (
  overrideResponse?:
    | TargetsProfileChange
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TargetsProfileChange> | TargetsProfileChange),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/profiles/:profileId",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateProfileResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getMakeDefaultProfileMockHandler = (
  overrideResponse?:
    | TargetsProfile
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsProfile> | TargetsProfile),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/profiles/:profileId/default",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getMakeDefaultProfileResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListQualitiesMockHandler = (
  overrideResponse?:
    | TargetsQualityList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsQualityList> | TargetsQualityList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/qualities",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListQualitiesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListTargetsMockHandler = (
  overrideResponse?:
    | TargetsTargetList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsTargetList> | TargetsTargetList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/targets",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListTargetsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateTargetMockHandler = (
  overrideResponse?:
    | TargetsTarget
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsTarget> | TargetsTarget),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/targets",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateTargetResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getDeleteTargetMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/targets/:targetId",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetTargetMockHandler = (
  overrideResponse?:
    | TargetsTarget
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsTarget> | TargetsTarget),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/targets/:targetId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetTargetResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateTargetMockHandler = (
  overrideResponse?:
    | TargetsTargetChange
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TargetsTargetChange> | TargetsTargetChange),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/targets/:targetId",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateTargetResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDuplicateTargetMockHandler = (
  overrideResponse?:
    | TargetsTarget
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<TargetsTarget> | TargetsTarget),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/targets/:targetId/duplicate",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getDuplicateTargetResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getGetWorkWantsMockHandler = (
  overrideResponse?:
    | TargetsWants
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<TargetsWants> | TargetsWants),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId/wants",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkWantsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetWorkWantsMockHandler = (
  overrideResponse?:
    | TargetsWantsChange
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<TargetsWantsChange> | TargetsWantsChange),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/:workId/wants",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorkWantsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getTargetsMock = () => [
  getListCutsMockHandler(),
  getCreateCutMockHandler(),
  getDeleteCutMockHandler(),
  getUpdateCutMockHandler(),
  getListProfilesMockHandler(),
  getCreateProfileMockHandler(),
  getDeleteProfileMockHandler(),
  getGetProfileMockHandler(),
  getUpdateProfileMockHandler(),
  getMakeDefaultProfileMockHandler(),
  getListQualitiesMockHandler(),
  getListTargetsMockHandler(),
  getCreateTargetMockHandler(),
  getDeleteTargetMockHandler(),
  getGetTargetMockHandler(),
  getUpdateTargetMockHandler(),
  getDuplicateTargetMockHandler(),
  getGetWorkWantsMockHandler(),
  getSetWorkWantsMockHandler(),
];
