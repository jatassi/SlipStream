import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  ImportsAddedList,
  ImportsAddedOutcome,
  ImportsExtensionImpact,
  ImportsFailedOutcome,
  ImportsImportEntry,
  ImportsImportingSettings,
  ImportsImportsList,
  ImportsList,
  ImportsSettings,
  SettingAddressField,
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingFolderField,
  SettingListField,
  SettingPortField,
  SettingRateField,
  SettingSizeField,
  SettingSpaceField,
  SettingSwitchField,
  SettingTemplateField,
  SettingWindowField,
} from "../../model";

export const getListImportsResponseMock = (
  overrideResponse: Partial<Extract<ImportsList, object>> = {},
): ImportsList => ({
  imports: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      edition: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      failure: faker.helpers.arrayElement([
        {
          code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
          sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
        null,
      ]),
      files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          actions: faker.helpers.arrayElements([
            "confirm",
            "retry",
            "decline",
            "force",
            "match",
            "notThisRelease",
            "changeEditions",
            "turnOnRenaming",
            "changeSetting",
          ] as const),
          cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
          fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => ({
              edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
              itemId: faker.string.uuid(),
              targetId: faker.string.uuid(),
            }),
          ),
          forces: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            cell: {
              edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
              itemId: faker.string.uuid(),
              targetId: faker.string.uuid(),
            },
            deletes: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          })),
          hold: faker.helpers.arrayElement([
            {
              attempts: faker.number.int({ min: 0 }),
              code: faker.helpers.arrayElement([
                "no_space",
                "root_not_ready",
                "node_away",
                "probe_timeout",
                "probe_error",
              ] as const),
              sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          itemIds: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          movable: faker.datatype.boolean(),
          outcome: faker.helpers.arrayElement([
            "pending",
            "left_out",
            "not_better",
            "waiting",
            "sending",
            "placing",
            "placed",
            "declined",
            "failed",
          ] as const),
          path: faker.string.alpha({ length: { min: 10, max: 20 } }),
          placedBy: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "link",
              "clone",
              "copy",
              "transfer",
              "move",
              "in_place",
            ] as const),
            null,
          ]),
          position: faker.number.int({ min: 0 }),
          quality: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
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
            null,
          ]),
          reason: faker.helpers.arrayElement([
            {
              code: faker.helpers.arrayElement([
                "sample",
                "bonus",
                "hidden_folder",
                "too_small",
                "extension",
                "runtime_sample",
                "sibling_better",
                "kept_better",
                "meets_nothing",
                "not_an_upgrade",
                "edition_not_wanted",
                "other_items",
                "no_number",
                "parts",
                "name_taken",
                "name_too_long",
                "settings",
                "busy",
                "confirm",
                "several",
                "no_match",
                "no_year",
                "elsewhere",
                "unlisted",
                "no_such_item",
                "several_items",
                "unreadable",
                "left",
                "no_rule",
                "import_failed",
                "root_changed",
                "by_admin",
                "removed",
                "canceled",
                "blocklisted",
                "work_deleted",
              ] as const),
              ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
              ...(faker.datatype.boolean()
                ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean()
                ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean()
                ? {
                    root: {
                      id: faker.string.uuid(),
                      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
                    },
                  }
                : {}),
              sentence: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              ...(faker.datatype.boolean()
                ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
            },
            null,
          ]),
          recommended: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "confirm",
              "retry",
              "decline",
              "force",
              "match",
              "notThisRelease",
              "changeEditions",
              "turnOnRenaming",
              "changeSetting",
            ] as const),
            null,
          ]),
          size: faker.number.int({ min: 0 }),
          staged: faker.number.int({ min: 0 }),
          verdicts: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            targetId: faker.string.uuid(),
          })),
        }),
      ),
      finishedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
      grab: faker.helpers.arrayElement([
        {
          id: faker.string.uuid(),
          release: faker.string.alpha({ length: { min: 10, max: 20 } }),
          stage: faker.helpers.arrayElement([
            "fetching",
            "downloading",
            "probing",
            "transferring",
            "importing",
            "imported",
            "failed",
          ] as const),
        },
        null,
      ]),
      hint: {
        ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
            value: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        year: faker.helpers.arrayElement([faker.number.int(), null]),
      },
      id: faker.string.uuid(),
      ignoredAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      moveFiles: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      root: faker.helpers.arrayElement([
        { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
        null,
      ]),
      seeding: faker.datatype.boolean(),
      source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
      stage: faker.helpers.arrayElement([
        "probing",
        "waiting",
        "placing",
        "done",
        "failed",
      ] as const),
      suggestion: faker.helpers.arrayElement([
        {
          confident: faker.datatype.boolean(),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          year: faker.helpers.arrayElement([faker.number.int(), null]),
        },
        null,
      ]),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    }),
  ),
  leads: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    added: faker.number.int({ min: 0 }),
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    needsYou: faker.number.int({ min: 0 }),
    root: { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
  })),
  next: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...overrideResponse,
});

export const getImportFolderResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportsList, object>> = {},
): ImportsImportsList =>
  faker.helpers.arrayElement([
    {
      imports: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
          createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          edition: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          failure: faker.helpers.arrayElement([
            {
              code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
              sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => ({
              actions: faker.helpers.arrayElements([
                "confirm",
                "retry",
                "decline",
                "force",
                "match",
                "notThisRelease",
                "changeEditions",
                "turnOnRenaming",
                "changeSetting",
              ] as const),
              cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
              fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
              fills: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => ({
                edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
                itemId: faker.string.uuid(),
                targetId: faker.string.uuid(),
              })),
              forces: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => ({
                cell: {
                  edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  itemId: faker.string.uuid(),
                  targetId: faker.string.uuid(),
                },
                deletes: Array.from(
                  { length: faker.number.int({ min: 1, max: 10 }) },
                  (_, i) => i + 1,
                ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              })),
              hold: faker.helpers.arrayElement([
                {
                  attempts: faker.number.int({ min: 0 }),
                  code: faker.helpers.arrayElement([
                    "no_space",
                    "root_not_ready",
                    "node_away",
                    "probe_timeout",
                    "probe_error",
                  ] as const),
                  sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
                },
                null,
              ]),
              itemIds: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              movable: faker.datatype.boolean(),
              outcome: faker.helpers.arrayElement([
                "pending",
                "left_out",
                "not_better",
                "waiting",
                "sending",
                "placing",
                "placed",
                "declined",
                "failed",
              ] as const),
              path: faker.string.alpha({ length: { min: 10, max: 20 } }),
              placedBy: faker.helpers.arrayElement([
                faker.helpers.arrayElement([
                  "link",
                  "clone",
                  "copy",
                  "transfer",
                  "move",
                  "in_place",
                ] as const),
                null,
              ]),
              position: faker.number.int({ min: 0 }),
              quality: faker.helpers.arrayElement([
                faker.helpers.arrayElement([
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
                null,
              ]),
              reason: faker.helpers.arrayElement([
                {
                  code: faker.helpers.arrayElement([
                    "sample",
                    "bonus",
                    "hidden_folder",
                    "too_small",
                    "extension",
                    "runtime_sample",
                    "sibling_better",
                    "kept_better",
                    "meets_nothing",
                    "not_an_upgrade",
                    "edition_not_wanted",
                    "other_items",
                    "no_number",
                    "parts",
                    "name_taken",
                    "name_too_long",
                    "settings",
                    "busy",
                    "confirm",
                    "several",
                    "no_match",
                    "no_year",
                    "elsewhere",
                    "unlisted",
                    "no_such_item",
                    "several_items",
                    "unreadable",
                    "left",
                    "no_rule",
                    "import_failed",
                    "root_changed",
                    "by_admin",
                    "removed",
                    "canceled",
                    "blocklisted",
                    "work_deleted",
                  ] as const),
                  ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
                  ...(faker.datatype.boolean()
                    ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                    : {}),
                  ...(faker.datatype.boolean()
                    ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                    : {}),
                  ...(faker.datatype.boolean()
                    ? {
                        root: {
                          id: faker.string.uuid(),
                          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
                        },
                      }
                    : {}),
                  sentence: faker.helpers.arrayElement([
                    faker.string.alpha({ length: { min: 10, max: 20 } }),
                    null,
                  ]),
                  ...(faker.datatype.boolean()
                    ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                    : {}),
                  ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
                },
                null,
              ]),
              recommended: faker.helpers.arrayElement([
                faker.helpers.arrayElement([
                  "confirm",
                  "retry",
                  "decline",
                  "force",
                  "match",
                  "notThisRelease",
                  "changeEditions",
                  "turnOnRenaming",
                  "changeSetting",
                ] as const),
                null,
              ]),
              size: faker.number.int({ min: 0 }),
              staged: faker.number.int({ min: 0 }),
              verdicts: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => ({
                sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
                targetId: faker.string.uuid(),
              })),
            }),
          ),
          finishedAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
          grab: faker.helpers.arrayElement([
            {
              id: faker.string.uuid(),
              release: faker.string.alpha({ length: { min: 10, max: 20 } }),
              stage: faker.helpers.arrayElement([
                "fetching",
                "downloading",
                "probing",
                "transferring",
                "importing",
                "imported",
                "failed",
              ] as const),
            },
            null,
          ]),
          hint: {
            ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
              () => ({
                provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
                value: faker.string.alpha({ length: { min: 10, max: 20 } }),
              }),
            ),
            title: faker.string.alpha({ length: { min: 10, max: 20 } }),
            year: faker.helpers.arrayElement([faker.number.int(), null]),
          },
          id: faker.string.uuid(),
          ignoredAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          moveFiles: faker.datatype.boolean(),
          nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          root: faker.helpers.arrayElement([
            {
              id: faker.string.uuid(),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          seeding: faker.datatype.boolean(),
          source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
          stage: faker.helpers.arrayElement([
            "probing",
            "waiting",
            "placing",
            "done",
            "failed",
          ] as const),
          suggestion: faker.helpers.arrayElement([
            {
              confident: faker.datatype.boolean(),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              posterUrl: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
              year: faker.helpers.arrayElement([faker.number.int(), null]),
            },
            null,
          ]),
          workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        }),
      ),
      ...overrideResponse,
    },
    {
      imports: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
          createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          edition: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          failure: faker.helpers.arrayElement([
            {
              code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
              sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => ({
              actions: faker.helpers.arrayElements([
                "confirm",
                "retry",
                "decline",
                "force",
                "match",
                "notThisRelease",
                "changeEditions",
                "turnOnRenaming",
                "changeSetting",
              ] as const),
              cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
              fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
              fills: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => ({
                edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
                itemId: faker.string.uuid(),
                targetId: faker.string.uuid(),
              })),
              forces: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => ({
                cell: {
                  edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
                  itemId: faker.string.uuid(),
                  targetId: faker.string.uuid(),
                },
                deletes: Array.from(
                  { length: faker.number.int({ min: 1, max: 10 }) },
                  (_, i) => i + 1,
                ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              })),
              hold: faker.helpers.arrayElement([
                {
                  attempts: faker.number.int({ min: 0 }),
                  code: faker.helpers.arrayElement([
                    "no_space",
                    "root_not_ready",
                    "node_away",
                    "probe_timeout",
                    "probe_error",
                  ] as const),
                  sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
                },
                null,
              ]),
              itemIds: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
              movable: faker.datatype.boolean(),
              outcome: faker.helpers.arrayElement([
                "pending",
                "left_out",
                "not_better",
                "waiting",
                "sending",
                "placing",
                "placed",
                "declined",
                "failed",
              ] as const),
              path: faker.string.alpha({ length: { min: 10, max: 20 } }),
              placedBy: faker.helpers.arrayElement([
                faker.helpers.arrayElement([
                  "link",
                  "clone",
                  "copy",
                  "transfer",
                  "move",
                  "in_place",
                ] as const),
                null,
              ]),
              position: faker.number.int({ min: 0 }),
              quality: faker.helpers.arrayElement([
                faker.helpers.arrayElement([
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
                null,
              ]),
              reason: faker.helpers.arrayElement([
                {
                  code: faker.helpers.arrayElement([
                    "sample",
                    "bonus",
                    "hidden_folder",
                    "too_small",
                    "extension",
                    "runtime_sample",
                    "sibling_better",
                    "kept_better",
                    "meets_nothing",
                    "not_an_upgrade",
                    "edition_not_wanted",
                    "other_items",
                    "no_number",
                    "parts",
                    "name_taken",
                    "name_too_long",
                    "settings",
                    "busy",
                    "confirm",
                    "several",
                    "no_match",
                    "no_year",
                    "elsewhere",
                    "unlisted",
                    "no_such_item",
                    "several_items",
                    "unreadable",
                    "left",
                    "no_rule",
                    "import_failed",
                    "root_changed",
                    "by_admin",
                    "removed",
                    "canceled",
                    "blocklisted",
                    "work_deleted",
                  ] as const),
                  ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
                  ...(faker.datatype.boolean()
                    ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                    : {}),
                  ...(faker.datatype.boolean()
                    ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                    : {}),
                  ...(faker.datatype.boolean()
                    ? {
                        root: {
                          id: faker.string.uuid(),
                          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
                        },
                      }
                    : {}),
                  sentence: faker.helpers.arrayElement([
                    faker.string.alpha({ length: { min: 10, max: 20 } }),
                    null,
                  ]),
                  ...(faker.datatype.boolean()
                    ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                    : {}),
                  ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
                },
                null,
              ]),
              recommended: faker.helpers.arrayElement([
                faker.helpers.arrayElement([
                  "confirm",
                  "retry",
                  "decline",
                  "force",
                  "match",
                  "notThisRelease",
                  "changeEditions",
                  "turnOnRenaming",
                  "changeSetting",
                ] as const),
                null,
              ]),
              size: faker.number.int({ min: 0 }),
              staged: faker.number.int({ min: 0 }),
              verdicts: Array.from(
                { length: faker.number.int({ min: 1, max: 10 }) },
                (_, i) => i + 1,
              ).map(() => ({
                sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
                targetId: faker.string.uuid(),
              })),
            }),
          ),
          finishedAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
          grab: faker.helpers.arrayElement([
            {
              id: faker.string.uuid(),
              release: faker.string.alpha({ length: { min: 10, max: 20 } }),
              stage: faker.helpers.arrayElement([
                "fetching",
                "downloading",
                "probing",
                "transferring",
                "importing",
                "imported",
                "failed",
              ] as const),
            },
            null,
          ]),
          hint: {
            ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
              () => ({
                provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
                value: faker.string.alpha({ length: { min: 10, max: 20 } }),
              }),
            ),
            title: faker.string.alpha({ length: { min: 10, max: 20 } }),
            year: faker.helpers.arrayElement([faker.number.int(), null]),
          },
          id: faker.string.uuid(),
          ignoredAt: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          moveFiles: faker.datatype.boolean(),
          nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          root: faker.helpers.arrayElement([
            {
              id: faker.string.uuid(),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          seeding: faker.datatype.boolean(),
          source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
          stage: faker.helpers.arrayElement([
            "probing",
            "waiting",
            "placing",
            "done",
            "failed",
          ] as const),
          suggestion: faker.helpers.arrayElement([
            {
              confident: faker.datatype.boolean(),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              posterUrl: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
              year: faker.helpers.arrayElement([faker.number.int(), null]),
            },
            null,
          ]),
          workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        }),
      ),
      ...overrideResponse,
    },
  ]);

export const getAddRootImportsResponseImportsAddedOutcomeMock = (
  overrideResponse: Partial<ImportsAddedOutcome> = {},
): ImportsAddedOutcome => ({
  ...{
    id: faker.string.uuid(),
    outcome: faker.helpers.arrayElement(["added"] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getAddRootImportsResponseImportsFailedOutcomeMock = (
  overrideResponse: Partial<ImportsFailedOutcome> = {},
): ImportsFailedOutcome => ({
  ...{
    id: faker.string.uuid(),
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
  },
  ...overrideResponse,
});

export const getAddRootImportsResponseMock = (
  overrideResponse: Partial<Extract<ImportsAddedList, object>> = {},
): ImportsAddedList => ({
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getAddRootImportsResponseImportsAddedOutcomeMock() },
      { ...getAddRootImportsResponseImportsFailedOutcomeMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getConfirmImportsResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportsList, object>> = {},
): ImportsImportsList => ({
  imports: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      edition: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      failure: faker.helpers.arrayElement([
        {
          code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
          sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
        null,
      ]),
      files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          actions: faker.helpers.arrayElements([
            "confirm",
            "retry",
            "decline",
            "force",
            "match",
            "notThisRelease",
            "changeEditions",
            "turnOnRenaming",
            "changeSetting",
          ] as const),
          cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
          fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
            () => ({
              edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
              itemId: faker.string.uuid(),
              targetId: faker.string.uuid(),
            }),
          ),
          forces: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            cell: {
              edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
              itemId: faker.string.uuid(),
              targetId: faker.string.uuid(),
            },
            deletes: Array.from(
              { length: faker.number.int({ min: 1, max: 10 }) },
              (_, i) => i + 1,
            ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          })),
          hold: faker.helpers.arrayElement([
            {
              attempts: faker.number.int({ min: 0 }),
              code: faker.helpers.arrayElement([
                "no_space",
                "root_not_ready",
                "node_away",
                "probe_timeout",
                "probe_error",
              ] as const),
              sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
            null,
          ]),
          itemIds: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => faker.string.alpha({ length: { min: 10, max: 20 } })),
          movable: faker.datatype.boolean(),
          outcome: faker.helpers.arrayElement([
            "pending",
            "left_out",
            "not_better",
            "waiting",
            "sending",
            "placing",
            "placed",
            "declined",
            "failed",
          ] as const),
          path: faker.string.alpha({ length: { min: 10, max: 20 } }),
          placedBy: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "link",
              "clone",
              "copy",
              "transfer",
              "move",
              "in_place",
            ] as const),
            null,
          ]),
          position: faker.number.int({ min: 0 }),
          quality: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
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
            null,
          ]),
          reason: faker.helpers.arrayElement([
            {
              code: faker.helpers.arrayElement([
                "sample",
                "bonus",
                "hidden_folder",
                "too_small",
                "extension",
                "runtime_sample",
                "sibling_better",
                "kept_better",
                "meets_nothing",
                "not_an_upgrade",
                "edition_not_wanted",
                "other_items",
                "no_number",
                "parts",
                "name_taken",
                "name_too_long",
                "settings",
                "busy",
                "confirm",
                "several",
                "no_match",
                "no_year",
                "elsewhere",
                "unlisted",
                "no_such_item",
                "several_items",
                "unreadable",
                "left",
                "no_rule",
                "import_failed",
                "root_changed",
                "by_admin",
                "removed",
                "canceled",
                "blocklisted",
                "work_deleted",
              ] as const),
              ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
              ...(faker.datatype.boolean()
                ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean()
                ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean()
                ? {
                    root: {
                      id: faker.string.uuid(),
                      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
                    },
                  }
                : {}),
              sentence: faker.helpers.arrayElement([
                faker.string.alpha({ length: { min: 10, max: 20 } }),
                null,
              ]),
              ...(faker.datatype.boolean()
                ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
                : {}),
              ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
            },
            null,
          ]),
          recommended: faker.helpers.arrayElement([
            faker.helpers.arrayElement([
              "confirm",
              "retry",
              "decline",
              "force",
              "match",
              "notThisRelease",
              "changeEditions",
              "turnOnRenaming",
              "changeSetting",
            ] as const),
            null,
          ]),
          size: faker.number.int({ min: 0 }),
          staged: faker.number.int({ min: 0 }),
          verdicts: Array.from(
            { length: faker.number.int({ min: 1, max: 10 }) },
            (_, i) => i + 1,
          ).map(() => ({
            sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
            targetId: faker.string.uuid(),
          })),
        }),
      ),
      finishedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
      grab: faker.helpers.arrayElement([
        {
          id: faker.string.uuid(),
          release: faker.string.alpha({ length: { min: 10, max: 20 } }),
          stage: faker.helpers.arrayElement([
            "fetching",
            "downloading",
            "probing",
            "transferring",
            "importing",
            "imported",
            "failed",
          ] as const),
        },
        null,
      ]),
      hint: {
        ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
            value: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        title: faker.string.alpha({ length: { min: 10, max: 20 } }),
        year: faker.helpers.arrayElement([faker.number.int(), null]),
      },
      id: faker.string.uuid(),
      ignoredAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      moveFiles: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      root: faker.helpers.arrayElement([
        { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
        null,
      ]),
      seeding: faker.datatype.boolean(),
      source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
      stage: faker.helpers.arrayElement([
        "probing",
        "waiting",
        "placing",
        "done",
        "failed",
      ] as const),
      suggestion: faker.helpers.arrayElement([
        {
          confident: faker.datatype.boolean(),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          posterUrl: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          year: faker.helpers.arrayElement([faker.number.int(), null]),
        },
        null,
      ]),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    }),
  ),
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
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

export const getGetImportingSettingsResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
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
    kind: faker.helpers.arrayElement(["count"] as const),
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
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingTemplateFieldMock = (
  overrideResponse: Partial<SettingTemplateField> = {},
): SettingTemplateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    field: faker.string.alpha({ length: { min: 10, max: 20 } }),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["template"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    plain: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingListFieldMock = (
  overrideResponse: Partial<SettingListField> = {},
): SettingListField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    item: faker.helpers.arrayElement(["extension"] as const),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["list"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    min: faker.number.int({ min: 0 }),
    pendingRestart: faker.datatype.boolean(),
    running: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseSettingSizeFieldMock = (
  overrideResponse: Partial<SettingSizeField> = {},
): SettingSizeField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["size"] as const),
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
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetImportingSettingsResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportingSettings, object>> = {},
): ImportsImportingSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetImportingSettingsResponseSettingDurationFieldMock() },
      { ...getGetImportingSettingsResponseSettingCountFieldMock() },
      { ...getGetImportingSettingsResponseSettingChoiceFieldMock() },
      { ...getGetImportingSettingsResponseSettingWindowFieldMock() },
      { ...getGetImportingSettingsResponseSettingPortFieldMock() },
      { ...getGetImportingSettingsResponseSettingSwitchFieldMock() },
      { ...getGetImportingSettingsResponseSettingRateFieldMock() },
      { ...getGetImportingSettingsResponseSettingAddressFieldMock() },
      { ...getGetImportingSettingsResponseSettingFolderFieldMock() },
      { ...getGetImportingSettingsResponseSettingSpaceFieldMock() },
      { ...getGetImportingSettingsResponseSettingTemplateFieldMock() },
      { ...getGetImportingSettingsResponseSettingListFieldMock() },
      { ...getGetImportingSettingsResponseSettingSizeFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
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

export const getUpdateImportingSettingsResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
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
    kind: faker.helpers.arrayElement(["count"] as const),
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
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingTemplateFieldMock = (
  overrideResponse: Partial<SettingTemplateField> = {},
): SettingTemplateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    field: faker.string.alpha({ length: { min: 10, max: 20 } }),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["template"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    plain: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingListFieldMock = (
  overrideResponse: Partial<SettingListField> = {},
): SettingListField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    item: faker.helpers.arrayElement(["extension"] as const),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["list"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    min: faker.number.int({ min: 0 }),
    pendingRestart: faker.datatype.boolean(),
    running: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseSettingSizeFieldMock = (
  overrideResponse: Partial<SettingSizeField> = {},
): SettingSizeField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["size"] as const),
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
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateImportingSettingsResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportingSettings, object>> = {},
): ImportsImportingSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateImportingSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingCountFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingPortFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingRateFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingSpaceFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingTemplateFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingListFieldMock() },
      { ...getUpdateImportingSettingsResponseSettingSizeFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getGetExtensionImpactResponseMock = (
  overrideResponse: Partial<Extract<ImportsExtensionImpact, object>> = {},
): ImportsExtensionImpact => ({
  files: faker.number.int({ min: 0 }),
  pending: faker.number.int({ min: 0 }),
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
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

export const getGetLibraryScanSettingsResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
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
    kind: faker.helpers.arrayElement(["count"] as const),
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
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingTemplateFieldMock = (
  overrideResponse: Partial<SettingTemplateField> = {},
): SettingTemplateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    field: faker.string.alpha({ length: { min: 10, max: 20 } }),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["template"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    plain: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingListFieldMock = (
  overrideResponse: Partial<SettingListField> = {},
): SettingListField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    item: faker.helpers.arrayElement(["extension"] as const),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["list"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    min: faker.number.int({ min: 0 }),
    pendingRestart: faker.datatype.boolean(),
    running: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseSettingSizeFieldMock = (
  overrideResponse: Partial<SettingSizeField> = {},
): SettingSizeField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["size"] as const),
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
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getGetLibraryScanSettingsResponseMock = (
  overrideResponse: Partial<Extract<ImportsSettings, object>> = {},
): ImportsSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetLibraryScanSettingsResponseSettingDurationFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingCountFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingChoiceFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingWindowFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingPortFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingSwitchFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingRateFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingAddressFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingFolderFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingSpaceFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingTemplateFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingListFieldMock() },
      { ...getGetLibraryScanSettingsResponseSettingSizeFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingDurationFieldMock = (
  overrideResponse: Partial<SettingDurationField> = {},
): SettingDurationField => ({
  ...{
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

export const getUpdateLibraryScanSettingsResponseSettingCountFieldMock = (
  overrideResponse: Partial<SettingCountField> = {},
): SettingCountField => ({
  ...{
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
    kind: faker.helpers.arrayElement(["count"] as const),
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
    unit: {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingChoiceFieldMock = (
  overrideResponse: Partial<SettingChoiceField> = {},
): SettingChoiceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["choice"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingWindowFieldMock = (
  overrideResponse: Partial<SettingWindowField> = {},
): SettingWindowField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["window"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      end: faker.string.alpha({ length: { min: 10, max: 20 } }),
      start: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    zone: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingPortFieldMock = (
  overrideResponse: Partial<SettingPortField> = {},
): SettingPortField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["port"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingSwitchFieldMock = (
  overrideResponse: Partial<SettingSwitchField> = {},
): SettingSwitchField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.datatype.boolean(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["switch"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.datatype.boolean(),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingRateFieldMock = (
  overrideResponse: Partial<SettingRateField> = {},
): SettingRateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["rate"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    max: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: faker.number.int(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { zero: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingAddressFieldMock = (
  overrideResponse: Partial<SettingAddressField> = {},
): SettingAddressField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["address"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingFolderFieldMock = (
  overrideResponse: Partial<SettingFolderField> = {},
): SettingFolderField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["folder"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    nodeId: faker.string.uuid(),
    pendingRestart: faker.datatype.boolean(),
    purpose: faker.helpers.arrayElement(["downloads"] as const),
    running: faker.string.alpha({ length: { min: 10, max: 20 } }),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingSpaceFieldMock = (
  overrideResponse: Partial<SettingSpaceField> = {},
): SettingSpaceField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["space"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    maxBytes: faker.number.int(),
    pendingRestart: faker.datatype.boolean(),
    running: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: {
      amount: faker.number.int({ min: 0 }),
      unit: faker.helpers.arrayElement(["bytes", "percent"] as const),
    },
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingTemplateFieldMock = (
  overrideResponse: Partial<SettingTemplateField> = {},
): SettingTemplateField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    field: faker.string.alpha({ length: { min: 10, max: 20 } }),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["template"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    pendingRestart: faker.datatype.boolean(),
    plain: faker.datatype.boolean(),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingListFieldMock = (
  overrideResponse: Partial<SettingListField> = {},
): SettingListField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    item: faker.helpers.arrayElement(["extension"] as const),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["list"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    ...(faker.datatype.boolean()
      ? { lockedBy: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    min: faker.number.int({ min: 0 }),
    pendingRestart: faker.datatype.boolean(),
    running: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    source: faker.helpers.arrayElement([
      "database",
      "default",
      "config.yaml",
      ".env",
      "env",
    ] as const),
    value: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseSettingSizeFieldMock = (
  overrideResponse: Partial<SettingSizeField> = {},
): SettingSizeField => ({
  ...{
    applies: faker.helpers.arrayElement(["live", "restart"] as const),
    default: faker.number.int(),
    ...(faker.datatype.boolean()
      ? { defaultFrom: faker.string.alpha({ length: { min: 10, max: 20 } }) }
      : {}),
    isDefault: faker.datatype.boolean(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["size"] as const),
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
    value: faker.number.int(),
  },
  ...overrideResponse,
});

export const getUpdateLibraryScanSettingsResponseMock = (
  overrideResponse: Partial<Extract<ImportsSettings, object>> = {},
): ImportsSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateLibraryScanSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingCountFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingPortFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingRateFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingSpaceFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingTemplateFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingListFieldMock() },
      { ...getUpdateLibraryScanSettingsResponseSettingSizeFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getGetImportResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportEntry, object>> = {},
): ImportsImportEntry => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  edition: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    actions: faker.helpers.arrayElements([
      "confirm",
      "retry",
      "decline",
      "force",
      "match",
      "notThisRelease",
      "changeEditions",
      "turnOnRenaming",
      "changeSetting",
    ] as const),
    cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        itemId: faker.string.uuid(),
        targetId: faker.string.uuid(),
      }),
    ),
    forces: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        cell: {
          edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
          itemId: faker.string.uuid(),
          targetId: faker.string.uuid(),
        },
        deletes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      }),
    ),
    hold: faker.helpers.arrayElement([
      {
        attempts: faker.number.int({ min: 0 }),
        code: faker.helpers.arrayElement([
          "no_space",
          "root_not_ready",
          "node_away",
          "probe_timeout",
          "probe_error",
        ] as const),
        sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    movable: faker.datatype.boolean(),
    outcome: faker.helpers.arrayElement([
      "pending",
      "left_out",
      "not_better",
      "waiting",
      "sending",
      "placing",
      "placed",
      "declined",
      "failed",
    ] as const),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    placedBy: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "link",
        "clone",
        "copy",
        "transfer",
        "move",
        "in_place",
      ] as const),
      null,
    ]),
    position: faker.number.int({ min: 0 }),
    quality: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
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
      null,
    ]),
    reason: faker.helpers.arrayElement([
      {
        code: faker.helpers.arrayElement([
          "sample",
          "bonus",
          "hidden_folder",
          "too_small",
          "extension",
          "runtime_sample",
          "sibling_better",
          "kept_better",
          "meets_nothing",
          "not_an_upgrade",
          "edition_not_wanted",
          "other_items",
          "no_number",
          "parts",
          "name_taken",
          "name_too_long",
          "settings",
          "busy",
          "confirm",
          "several",
          "no_match",
          "no_year",
          "elsewhere",
          "unlisted",
          "no_such_item",
          "several_items",
          "unreadable",
          "left",
          "no_rule",
          "import_failed",
          "root_changed",
          "by_admin",
          "removed",
          "canceled",
          "blocklisted",
          "work_deleted",
        ] as const),
        ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
        ...(faker.datatype.boolean()
          ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? {
              root: {
                id: faker.string.uuid(),
                label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
            }
          : {}),
        sentence: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        ...(faker.datatype.boolean()
          ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
      },
      null,
    ]),
    recommended: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "confirm",
        "retry",
        "decline",
        "force",
        "match",
        "notThisRelease",
        "changeEditions",
        "turnOnRenaming",
        "changeSetting",
      ] as const),
      null,
    ]),
    size: faker.number.int({ min: 0 }),
    staged: faker.number.int({ min: 0 }),
    verdicts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        targetId: faker.string.uuid(),
      }),
    ),
  })),
  finishedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  grab: faker.helpers.arrayElement([
    {
      id: faker.string.uuid(),
      release: faker.string.alpha({ length: { min: 10, max: 20 } }),
      stage: faker.helpers.arrayElement([
        "fetching",
        "downloading",
        "probing",
        "transferring",
        "importing",
        "imported",
        "failed",
      ] as const),
    },
    null,
  ]),
  hint: {
    ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
      provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
      value: faker.string.alpha({ length: { min: 10, max: 20 } }),
    })),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    year: faker.helpers.arrayElement([faker.number.int(), null]),
  },
  id: faker.string.uuid(),
  ignoredAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  moveFiles: faker.datatype.boolean(),
  nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  root: faker.helpers.arrayElement([
    { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  seeding: faker.datatype.boolean(),
  source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
  stage: faker.helpers.arrayElement(["probing", "waiting", "placing", "done", "failed"] as const),
  suggestion: faker.helpers.arrayElement([
    {
      confident: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      posterUrl: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      year: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...overrideResponse,
});

export const getActOnRootImportResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportEntry, object>> = {},
): ImportsImportEntry => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  edition: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    actions: faker.helpers.arrayElements([
      "confirm",
      "retry",
      "decline",
      "force",
      "match",
      "notThisRelease",
      "changeEditions",
      "turnOnRenaming",
      "changeSetting",
    ] as const),
    cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        itemId: faker.string.uuid(),
        targetId: faker.string.uuid(),
      }),
    ),
    forces: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        cell: {
          edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
          itemId: faker.string.uuid(),
          targetId: faker.string.uuid(),
        },
        deletes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      }),
    ),
    hold: faker.helpers.arrayElement([
      {
        attempts: faker.number.int({ min: 0 }),
        code: faker.helpers.arrayElement([
          "no_space",
          "root_not_ready",
          "node_away",
          "probe_timeout",
          "probe_error",
        ] as const),
        sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    movable: faker.datatype.boolean(),
    outcome: faker.helpers.arrayElement([
      "pending",
      "left_out",
      "not_better",
      "waiting",
      "sending",
      "placing",
      "placed",
      "declined",
      "failed",
    ] as const),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    placedBy: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "link",
        "clone",
        "copy",
        "transfer",
        "move",
        "in_place",
      ] as const),
      null,
    ]),
    position: faker.number.int({ min: 0 }),
    quality: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
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
      null,
    ]),
    reason: faker.helpers.arrayElement([
      {
        code: faker.helpers.arrayElement([
          "sample",
          "bonus",
          "hidden_folder",
          "too_small",
          "extension",
          "runtime_sample",
          "sibling_better",
          "kept_better",
          "meets_nothing",
          "not_an_upgrade",
          "edition_not_wanted",
          "other_items",
          "no_number",
          "parts",
          "name_taken",
          "name_too_long",
          "settings",
          "busy",
          "confirm",
          "several",
          "no_match",
          "no_year",
          "elsewhere",
          "unlisted",
          "no_such_item",
          "several_items",
          "unreadable",
          "left",
          "no_rule",
          "import_failed",
          "root_changed",
          "by_admin",
          "removed",
          "canceled",
          "blocklisted",
          "work_deleted",
        ] as const),
        ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
        ...(faker.datatype.boolean()
          ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? {
              root: {
                id: faker.string.uuid(),
                label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
            }
          : {}),
        sentence: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        ...(faker.datatype.boolean()
          ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
      },
      null,
    ]),
    recommended: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "confirm",
        "retry",
        "decline",
        "force",
        "match",
        "notThisRelease",
        "changeEditions",
        "turnOnRenaming",
        "changeSetting",
      ] as const),
      null,
    ]),
    size: faker.number.int({ min: 0 }),
    staged: faker.number.int({ min: 0 }),
    verdicts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        targetId: faker.string.uuid(),
      }),
    ),
  })),
  finishedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  grab: faker.helpers.arrayElement([
    {
      id: faker.string.uuid(),
      release: faker.string.alpha({ length: { min: 10, max: 20 } }),
      stage: faker.helpers.arrayElement([
        "fetching",
        "downloading",
        "probing",
        "transferring",
        "importing",
        "imported",
        "failed",
      ] as const),
    },
    null,
  ]),
  hint: {
    ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
      provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
      value: faker.string.alpha({ length: { min: 10, max: 20 } }),
    })),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    year: faker.helpers.arrayElement([faker.number.int(), null]),
  },
  id: faker.string.uuid(),
  ignoredAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  moveFiles: faker.datatype.boolean(),
  nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  root: faker.helpers.arrayElement([
    { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  seeding: faker.datatype.boolean(),
  source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
  stage: faker.helpers.arrayElement(["probing", "waiting", "placing", "done", "failed"] as const),
  suggestion: faker.helpers.arrayElement([
    {
      confident: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      posterUrl: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      year: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...overrideResponse,
});

export const getActOnImportFileResponseMock = (
  overrideResponse: Partial<Extract<ImportsImportEntry, object>> = {},
): ImportsImportEntry => ({
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  edition: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  failure: faker.helpers.arrayElement([
    {
      code: faker.helpers.arrayElement(["no_video_file", "unreadable"] as const),
      sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  files: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    actions: faker.helpers.arrayElements([
      "confirm",
      "retry",
      "decline",
      "force",
      "match",
      "notThisRelease",
      "changeEditions",
      "turnOnRenaming",
      "changeSetting",
    ] as const),
    cut: faker.string.alpha({ length: { min: 10, max: 20 } }),
    fileId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    fills: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
        itemId: faker.string.uuid(),
        targetId: faker.string.uuid(),
      }),
    ),
    forces: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        cell: {
          edition: faker.string.alpha({ length: { min: 10, max: 20 } }),
          itemId: faker.string.uuid(),
          targetId: faker.string.uuid(),
        },
        deletes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      }),
    ),
    hold: faker.helpers.arrayElement([
      {
        attempts: faker.number.int({ min: 0 }),
        code: faker.helpers.arrayElement([
          "no_space",
          "root_not_ready",
          "node_away",
          "probe_timeout",
          "probe_error",
        ] as const),
        sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    itemIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
    movable: faker.datatype.boolean(),
    outcome: faker.helpers.arrayElement([
      "pending",
      "left_out",
      "not_better",
      "waiting",
      "sending",
      "placing",
      "placed",
      "declined",
      "failed",
    ] as const),
    path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    placedBy: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "link",
        "clone",
        "copy",
        "transfer",
        "move",
        "in_place",
      ] as const),
      null,
    ]),
    position: faker.number.int({ min: 0 }),
    quality: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
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
      null,
    ]),
    reason: faker.helpers.arrayElement([
      {
        code: faker.helpers.arrayElement([
          "sample",
          "bonus",
          "hidden_folder",
          "too_small",
          "extension",
          "runtime_sample",
          "sibling_better",
          "kept_better",
          "meets_nothing",
          "not_an_upgrade",
          "edition_not_wanted",
          "other_items",
          "no_number",
          "parts",
          "name_taken",
          "name_too_long",
          "settings",
          "busy",
          "confirm",
          "several",
          "no_match",
          "no_year",
          "elsewhere",
          "unlisted",
          "no_such_item",
          "several_items",
          "unreadable",
          "left",
          "no_rule",
          "import_failed",
          "root_changed",
          "by_admin",
          "removed",
          "canceled",
          "blocklisted",
          "work_deleted",
        ] as const),
        ...(faker.datatype.boolean() ? { count: faker.number.int() } : {}),
        ...(faker.datatype.boolean()
          ? { line: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? { names: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean()
          ? {
              root: {
                id: faker.string.uuid(),
                label: faker.string.alpha({ length: { min: 10, max: 20 } }),
              },
            }
          : {}),
        sentence: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        ...(faker.datatype.boolean()
          ? { title: faker.string.alpha({ length: { min: 10, max: 20 } }) }
          : {}),
        ...(faker.datatype.boolean() ? { year: faker.number.int() } : {}),
      },
      null,
    ]),
    recommended: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "confirm",
        "retry",
        "decline",
        "force",
        "match",
        "notThisRelease",
        "changeEditions",
        "turnOnRenaming",
        "changeSetting",
      ] as const),
      null,
    ]),
    size: faker.number.int({ min: 0 }),
    staged: faker.number.int({ min: 0 }),
    verdicts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        sentence: faker.string.alpha({ length: { min: 10, max: 20 } }),
        targetId: faker.string.uuid(),
      }),
    ),
  })),
  finishedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
  grab: faker.helpers.arrayElement([
    {
      id: faker.string.uuid(),
      release: faker.string.alpha({ length: { min: 10, max: 20 } }),
      stage: faker.helpers.arrayElement([
        "fetching",
        "downloading",
        "probing",
        "transferring",
        "importing",
        "imported",
        "failed",
      ] as const),
    },
    null,
  ]),
  hint: {
    ids: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
      provider: faker.helpers.arrayElement(["tmdb", "imdb", "tvdb"] as const),
      value: faker.string.alpha({ length: { min: 10, max: 20 } }),
    })),
    title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    year: faker.helpers.arrayElement([faker.number.int(), null]),
  },
  id: faker.string.uuid(),
  ignoredAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  moveFiles: faker.datatype.boolean(),
  nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  root: faker.helpers.arrayElement([
    { id: faker.string.uuid(), label: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    null,
  ]),
  seeding: faker.datatype.boolean(),
  source: faker.helpers.arrayElement(["grab", "root", "folder"] as const),
  stage: faker.helpers.arrayElement(["probing", "waiting", "placing", "done", "failed"] as const),
  suggestion: faker.helpers.arrayElement([
    {
      confident: faker.datatype.boolean(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      posterUrl: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      year: faker.helpers.arrayElement([faker.number.int(), null]),
    },
    null,
  ]),
  workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
  ...overrideResponse,
});

export const getListImportsMockHandler = (
  overrideResponse?:
    | ImportsList
    | ((info: Parameters<Parameters<typeof http.get>[1]>[0]) => Promise<ImportsList> | ImportsList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/imports",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListImportsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getImportFolderMockHandler = (
  overrideResponse?:
    | ImportsImportsList
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ImportsImportsList> | ImportsImportsList),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getImportFolderResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddRootImportsMockHandler = (
  overrideResponse?:
    | ImportsAddedList
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ImportsAddedList> | ImportsAddedList),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/add",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddRootImportsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getConfirmImportsMockHandler = (
  overrideResponse?:
    | ImportsImportsList
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ImportsImportsList> | ImportsImportsList),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/confirm",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getConfirmImportsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getIgnoreImportsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/ignore",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetImportingSettingsMockHandler = (
  overrideResponse?:
    | ImportsImportingSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ImportsImportingSettings> | ImportsImportingSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/imports/importing-settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetImportingSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateImportingSettingsMockHandler = (
  overrideResponse?:
    | ImportsImportingSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<ImportsImportingSettings> | ImportsImportingSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/imports/importing-settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateImportingSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetExtensionImpactMockHandler = (
  overrideResponse?:
    | ImportsExtensionImpact
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ImportsExtensionImpact> | ImportsExtensionImpact),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/imports/importing-settings/extension-impact",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetExtensionImpactResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetImportMoveFilesMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/move-files",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetLibraryScanSettingsMockHandler = (
  overrideResponse?:
    | ImportsSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ImportsSettings> | ImportsSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/imports/scan-settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetLibraryScanSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateLibraryScanSettingsMockHandler = (
  overrideResponse?:
    | ImportsSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<ImportsSettings> | ImportsSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/imports/scan-settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateLibraryScanSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUnignoreImportsMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/unignore",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetImportMockHandler = (
  overrideResponse?:
    | ImportsImportEntry
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ImportsImportEntry> | ImportsImportEntry),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/imports/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetImportResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getActOnRootImportMockHandler = (
  overrideResponse?:
    | ImportsImportEntry
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ImportsImportEntry> | ImportsImportEntry),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/:id",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getActOnRootImportResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getActOnImportFileMockHandler = (
  overrideResponse?:
    | ImportsImportEntry
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<ImportsImportEntry> | ImportsImportEntry),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/imports/:id/files/:position",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getActOnImportFileResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getImportsMock = () => [
  getListImportsMockHandler(),
  getImportFolderMockHandler(),
  getAddRootImportsMockHandler(),
  getConfirmImportsMockHandler(),
  getIgnoreImportsMockHandler(),
  getGetImportingSettingsMockHandler(),
  getUpdateImportingSettingsMockHandler(),
  getGetExtensionImpactMockHandler(),
  getSetImportMoveFilesMockHandler(),
  getGetLibraryScanSettingsMockHandler(),
  getUpdateLibraryScanSettingsMockHandler(),
  getUnignoreImportsMockHandler(),
  getGetImportMockHandler(),
  getActOnRootImportMockHandler(),
  getActOnImportFileMockHandler(),
];
