import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
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
  WorksAdded,
  WorksCandidateList,
  WorksContentType,
  WorksContentTypeList,
  WorksDateValue,
  WorksDeleted,
  WorksDoneOutcome,
  WorksDurationValue,
  WorksFailedOutcome,
  WorksLibraryRows,
  WorksLinkValue,
  WorksMonitoringDoneOutcome,
  WorksMonitoringList,
  WorksMonitoringSkippedOutcome,
  WorksNumberValue,
  WorksOptionsChanged,
  WorksOutcomeList,
  WorksPeopleValue,
  WorksPreviewed,
  WorksProfileChanged,
  WorksSearchResult,
  WorksSkippedOutcome,
  WorksTextValue,
  WorksTextsValue,
  WorksTypeSettings,
  WorksWorkView,
} from "../../model";

export const getListContentTypesResponseMock = (
  overrideResponse: Partial<Extract<WorksContentTypeList, object>> = {},
): WorksContentTypeList => ({
  contentTypes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      art: {
        backdrop: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        poster: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        title: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
      },
      artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          shape: faker.helpers.arrayElement([
            "poster",
            "backdrop",
            "logo",
            "photo",
            "cover",
          ] as const),
          tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
        }),
      ),
      aside: faker.helpers.arrayElement([
        {
          many: faker.string.alpha({ length: { min: 10, max: 20 } }),
          one: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
        null,
      ]),
      availability: faker.string.alpha({ length: { min: 10, max: 20 } }),
      enabled: faker.datatype.boolean(),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          art: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
          in: faker.helpers.arrayElements([
            "facts",
            "tagline",
            "overview",
            "about",
            "credits",
            "details",
            "code",
          ] as const),
          key: faker.string.alpha({ length: { min: 10, max: 20 } }),
          kind: faker.helpers.arrayElement([
            "text",
            "number",
            "date",
            "duration",
            "texts",
            "people",
            "link",
          ] as const),
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          list: {
            column: faker.datatype.boolean(),
            filter: faker.datatype.boolean(),
            sort: faker.datatype.boolean(),
          },
          tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
          with: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
        }),
      ),
      icon: faker.helpers.arrayElement([
        "film",
        "tv",
        "note",
        "open-book",
        "headphones",
        "panels",
        "image",
        "shapes",
      ] as const),
      id: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
      item: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      items: faker.helpers.arrayElement(["one", "many"] as const),
      levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          many: faker.string.alpha({ length: { min: 10, max: 20 } }),
          one: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      releases: faker.string.alpha({ length: { min: 10, max: 20 } }),
      source: faker.string.alpha({ length: { min: 10, max: 20 } }),
      work: {
        many: faker.string.alpha({ length: { min: 10, max: 20 } }),
        one: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
    }),
  ),
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        photoUrl: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getSearchCatalogResponseMock = (
  overrideResponse: Partial<Extract<WorksCandidateList, object>> = {},
): WorksCandidateList => ({
  candidates: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getSearchCatalogResponseWorksTextValueMock() },
            { ...getSearchCatalogResponseWorksNumberValueMock() },
            { ...getSearchCatalogResponseWorksDateValueMock() },
            { ...getSearchCatalogResponseWorksDurationValueMock() },
            { ...getSearchCatalogResponseWorksTextsValueMock() },
            { ...getSearchCatalogResponseWorksPeopleValueMock() },
            { ...getSearchCatalogResponseWorksLinkValueMock() },
          ]),
      ),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      original: faker.string.alpha({ length: { min: 10, max: 20 } }),
      overview: faker.string.alpha({ length: { min: 10, max: 20 } }),
      posterUrl: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      unlisted: faker.datatype.boolean(),
      workId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    }),
  ),
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        photoUrl: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getPreviewCatalogRecordResponseMock = (
  overrideResponse: Partial<Extract<WorksPreviewed, object>> = {},
): WorksPreviewed => ({
  inLibrary: faker.datatype.boolean(),
  presets: faker.helpers.arrayElement([
    {
      asides: faker.number.int(),
      choices: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          monitors: faker.number.int(),
          monitorsNew: faker.datatype.boolean(),
          preset: faker.helpers.arrayElement([
            "all",
            "future",
            "missing",
            "existing",
            "first",
            "latest",
            "none",
          ] as const),
        }),
      ),
    },
    null,
  ]),
  work: {
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
        url: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getPreviewCatalogRecordResponseWorksTextValueMock() },
        { ...getPreviewCatalogRecordResponseWorksNumberValueMock() },
        { ...getPreviewCatalogRecordResponseWorksDateValueMock() },
        { ...getPreviewCatalogRecordResponseWorksDurationValueMock() },
        { ...getPreviewCatalogRecordResponseWorksTextsValueMock() },
        { ...getPreviewCatalogRecordResponseWorksPeopleValueMock() },
        { ...getPreviewCatalogRecordResponseWorksLinkValueMock() },
      ]),
    ),
    goneUpstreamAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    id: faker.string.uuid(),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            url: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        aside: faker.datatype.boolean(),
        availability: faker.helpers.arrayElement([
          {
            basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
            date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
          },
          null,
        ]),
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getPreviewCatalogRecordResponseWorksTextValueMock() },
              { ...getPreviewCatalogRecordResponseWorksNumberValueMock() },
              { ...getPreviewCatalogRecordResponseWorksDateValueMock() },
              { ...getPreviewCatalogRecordResponseWorksDurationValueMock() },
              { ...getPreviewCatalogRecordResponseWorksTextsValueMock() },
              { ...getPreviewCatalogRecordResponseWorksPeopleValueMock() },
              { ...getPreviewCatalogRecordResponseWorksLinkValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        monitored: faker.datatype.boolean(),
        position: faker.number.int({ min: 1 }),
        ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            scale: faker.number.float({ min: 0, fractionDigits: 2 }),
            source: faker.string.alpha({ length: { min: 10, max: 20 } }),
            value: faker.number.float({ min: 0, fractionDigits: 2 }),
            votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
          }),
        ),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            url: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        aside: faker.datatype.boolean(),
        depth: faker.number.int({ min: 1 }),
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getPreviewCatalogRecordResponseWorksTextValueMock() },
              { ...getPreviewCatalogRecordResponseWorksNumberValueMock() },
              { ...getPreviewCatalogRecordResponseWorksDateValueMock() },
              { ...getPreviewCatalogRecordResponseWorksDurationValueMock() },
              { ...getPreviewCatalogRecordResponseWorksTextsValueMock() },
              { ...getPreviewCatalogRecordResponseWorksPeopleValueMock() },
              { ...getPreviewCatalogRecordResponseWorksLinkValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        position: faker.number.int({ min: 1 }),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
        votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      }),
    ),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getSetContentTypeEnabledResponseMock = (
  overrideResponse: Partial<Extract<WorksContentType, object>> = {},
): WorksContentType => ({
  art: {
    backdrop: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    poster: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    title: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  },
  artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      shape: faker.helpers.arrayElement(["poster", "backdrop", "logo", "photo", "cover"] as const),
      tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
    }),
  ),
  aside: faker.helpers.arrayElement([
    {
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  availability: faker.string.alpha({ length: { min: 10, max: 20 } }),
  enabled: faker.datatype.boolean(),
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      art: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
      ...(faker.datatype.boolean() ? { depth: faker.number.int({ min: 1 }) } : {}),
      in: faker.helpers.arrayElements([
        "facts",
        "tagline",
        "overview",
        "about",
        "credits",
        "details",
        "code",
      ] as const),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.helpers.arrayElement([
        "text",
        "number",
        "date",
        "duration",
        "texts",
        "people",
        "link",
      ] as const),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      list: {
        column: faker.datatype.boolean(),
        filter: faker.datatype.boolean(),
        sort: faker.datatype.boolean(),
      },
      tier: faker.helpers.arrayElement(["work", "level", "item"] as const),
      with: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
    }),
  ),
  icon: faker.helpers.arrayElement([
    "film",
    "tv",
    "note",
    "open-book",
    "headphones",
    "panels",
    "image",
    "shapes",
  ] as const),
  id: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  item: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  items: faker.helpers.arrayElement(["one", "many"] as const),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      many: faker.string.alpha({ length: { min: 10, max: 20 } }),
      one: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  releases: faker.string.alpha({ length: { min: 10, max: 20 } }),
  source: faker.string.alpha({ length: { min: 10, max: 20 } }),
  work: {
    many: faker.string.alpha({ length: { min: 10, max: 20 } }),
    one: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetContentTypeSettingsResponseSettingDurationFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingCountFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingChoiceFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingWindowFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingPortFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingSwitchFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingRateFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingAddressFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingFolderFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingSpaceFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingTemplateFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingListFieldMock = (
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

export const getGetContentTypeSettingsResponseSettingSizeFieldMock = (
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

export const getGetContentTypeSettingsResponseMock = (
  overrideResponse: Partial<Extract<WorksTypeSettings, object>> = {},
): WorksTypeSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetContentTypeSettingsResponseSettingDurationFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingCountFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingChoiceFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingWindowFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingPortFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingSwitchFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingRateFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingAddressFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingFolderFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingSpaceFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingTemplateFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingListFieldMock() },
      { ...getGetContentTypeSettingsResponseSettingSizeFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getUpdateContentTypeSettingsResponseSettingDurationFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingCountFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingChoiceFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingWindowFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingPortFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingSwitchFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingRateFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingAddressFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingFolderFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingSpaceFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingTemplateFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingListFieldMock = (
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

export const getUpdateContentTypeSettingsResponseSettingSizeFieldMock = (
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

export const getUpdateContentTypeSettingsResponseMock = (
  overrideResponse: Partial<Extract<WorksTypeSettings, object>> = {},
): WorksTypeSettings => ({
  settings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getUpdateContentTypeSettingsResponseSettingDurationFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingCountFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingChoiceFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingWindowFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingPortFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingSwitchFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingRateFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingAddressFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingFolderFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingSpaceFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingTemplateFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingListFieldMock() },
      { ...getUpdateContentTypeSettingsResponseSettingSizeFieldMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getListLibraryResponseMock = (
  overrideResponse: Partial<Extract<WorksLibraryRows, object>> = {},
): WorksLibraryRows => ({
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    addedBy: faker.helpers.arrayElement(["admin", "scan"] as const),
    availableOn: faker.helpers.arrayElement([
      faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
      null,
    ]),
    bytes: faker.number.int({ min: 0 }),
    cells: {
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    files: faker.number.int({ min: 0 }),
    goneFiles: faker.number.int({ min: 0 }),
    goneUpstreamAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    id: faker.string.uuid(),
    itemStatus: {
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    items: faker.number.int({ min: 0 }),
    kept: faker.number.int({ min: 0 }),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    monitored: faker.helpers.arrayElement(["all", "some", "none"] as const),
    needsYou: faker.number.int({ min: 0 }),
    posterUrl: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    profileId: faker.string.uuid(),
    rating: faker.helpers.arrayElement([
      {
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
      },
      null,
    ]),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    rootId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
    status: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "failed",
        "missing",
        "downloading",
        "upgradable",
        "unreleased",
        "met",
      ] as const),
      null,
    ]),
    values: {
      [faker.string.alphanumeric(5)]: faker.helpers.arrayElement([
        faker.number.float({ fractionDigits: 2 }),
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      ]),
    },
  })),
  ...overrideResponse,
});

export const getAddWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        photoUrl: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getAddWorkResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getAddWorkResponseSettingDurationFieldMock = (
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

export const getAddWorkResponseSettingCountFieldMock = (
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

export const getAddWorkResponseSettingChoiceFieldMock = (
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

export const getAddWorkResponseSettingWindowFieldMock = (
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

export const getAddWorkResponseSettingPortFieldMock = (
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

export const getAddWorkResponseSettingSwitchFieldMock = (
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

export const getAddWorkResponseSettingRateFieldMock = (
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

export const getAddWorkResponseSettingAddressFieldMock = (
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

export const getAddWorkResponseSettingFolderFieldMock = (
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

export const getAddWorkResponseSettingSpaceFieldMock = (
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

export const getAddWorkResponseSettingTemplateFieldMock = (
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

export const getAddWorkResponseSettingListFieldMock = (
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

export const getAddWorkResponseSettingSizeFieldMock = (
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

export const getAddWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksAdded, object>> = {},
): WorksAdded => ({
  created: faker.datatype.boolean(),
  work: {
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
        url: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getAddWorkResponseWorksTextValueMock() },
        { ...getAddWorkResponseWorksNumberValueMock() },
        { ...getAddWorkResponseWorksDateValueMock() },
        { ...getAddWorkResponseWorksDurationValueMock() },
        { ...getAddWorkResponseWorksTextsValueMock() },
        { ...getAddWorkResponseWorksPeopleValueMock() },
        { ...getAddWorkResponseWorksLinkValueMock() },
      ]),
    ),
    goneUpstreamAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    id: faker.string.uuid(),
    items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            url: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        aside: faker.datatype.boolean(),
        availability: faker.helpers.arrayElement([
          {
            basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
            date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
          },
          null,
        ]),
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getAddWorkResponseWorksTextValueMock() },
              { ...getAddWorkResponseWorksNumberValueMock() },
              { ...getAddWorkResponseWorksDateValueMock() },
              { ...getAddWorkResponseWorksDurationValueMock() },
              { ...getAddWorkResponseWorksTextsValueMock() },
              { ...getAddWorkResponseWorksPeopleValueMock() },
              { ...getAddWorkResponseWorksLinkValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        monitored: faker.datatype.boolean(),
        position: faker.number.int({ min: 1 }),
        ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
            label: faker.string.alpha({ length: { min: 10, max: 20 } }),
            scale: faker.number.float({ min: 0, fractionDigits: 2 }),
            source: faker.string.alpha({ length: { min: 10, max: 20 } }),
            value: faker.number.float({ min: 0, fractionDigits: 2 }),
            votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
          }),
        ),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levelFolders: faker.datatype.boolean(),
    levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
            url: faker.string.alpha({ length: { min: 10, max: 20 } }),
          }),
        ),
        aside: faker.datatype.boolean(),
        depth: faker.number.int({ min: 1 }),
        fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () =>
            faker.helpers.arrayElement([
              { ...getAddWorkResponseWorksTextValueMock() },
              { ...getAddWorkResponseWorksNumberValueMock() },
              { ...getAddWorkResponseWorksDateValueMock() },
              { ...getAddWorkResponseWorksDurationValueMock() },
              { ...getAddWorkResponseWorksTextsValueMock() },
              { ...getAddWorkResponseWorksPeopleValueMock() },
              { ...getAddWorkResponseWorksLinkValueMock() },
            ]),
        ),
        id: faker.string.uuid(),
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        position: faker.number.int({ min: 1 }),
        withdrawnAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      }),
    ),
    links: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        key: faker.string.alpha({ length: { min: 10, max: 20 } }),
        kind: faker.helpers.arrayElement(["page", "trailer"] as const),
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        url: faker.internet.url(),
      }),
    ),
    monitorAsides: faker.datatype.boolean(),
    monitorNew: faker.datatype.boolean(),
    openIn: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        kind: faker.helpers.arrayElement(["jellyfin", "emby"] as const),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        serverId: faker.string.uuid(),
        url: faker.internet.url(),
      }),
    ),
    options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () =>
        faker.helpers.arrayElement([
          { ...getAddWorkResponseSettingDurationFieldMock() },
          { ...getAddWorkResponseSettingCountFieldMock() },
          { ...getAddWorkResponseSettingChoiceFieldMock() },
          { ...getAddWorkResponseSettingWindowFieldMock() },
          { ...getAddWorkResponseSettingPortFieldMock() },
          { ...getAddWorkResponseSettingSwitchFieldMock() },
          { ...getAddWorkResponseSettingRateFieldMock() },
          { ...getAddWorkResponseSettingAddressFieldMock() },
          { ...getAddWorkResponseSettingFolderFieldMock() },
          { ...getAddWorkResponseSettingSpaceFieldMock() },
          { ...getAddWorkResponseSettingTemplateFieldMock() },
          { ...getAddWorkResponseSettingListFieldMock() },
          { ...getAddWorkResponseSettingSizeFieldMock() },
        ]),
    ),
    profile: {
      id: faker.string.uuid(),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
        votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      }),
    ),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    root: faker.helpers.arrayElement([
      {
        id: faker.string.uuid(),
        nodeId: faker.string.uuid(),
        path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getDeleteWorksResponseWorksDoneOutcomeMock = (
  overrideResponse: Partial<WorksDoneOutcome> = {},
): WorksDoneOutcome => ({
  ...{ outcome: faker.helpers.arrayElement(["done"] as const), workId: faker.string.uuid() },
  ...overrideResponse,
});

export const getDeleteWorksResponseWorksSkippedOutcomeMock = (
  overrideResponse: Partial<WorksSkippedOutcome> = {},
): WorksSkippedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["skipped"] as const),
    reason: faker.helpers.arrayElement([
      "already",
      "work_has_files",
      "work_gone_upstream",
      "work_not_found",
    ] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getDeleteWorksResponseWorksFailedOutcomeMock = (
  overrideResponse: Partial<WorksFailedOutcome> = {},
): WorksFailedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getDeleteWorksResponseMock = (
  overrideResponse: Partial<Extract<WorksDeleted, object>> = {},
): WorksDeleted => ({
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getDeleteWorksResponseWorksDoneOutcomeMock() },
      { ...getDeleteWorksResponseWorksSkippedOutcomeMock() },
      { ...getDeleteWorksResponseWorksFailedOutcomeMock() },
    ]),
  ),
  plan: {
    bytes: faker.number.int({ min: 0 }),
    downloading: faker.number.int({ min: 0 }),
    files: faker.number.int({ min: 0 }),
    kept: faker.number.int({ min: 0 }),
    works: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getSetWorksMonitoredResponseWorksDoneOutcomeMock = (
  overrideResponse: Partial<WorksDoneOutcome> = {},
): WorksDoneOutcome => ({
  ...{ outcome: faker.helpers.arrayElement(["done"] as const), workId: faker.string.uuid() },
  ...overrideResponse,
});

export const getSetWorksMonitoredResponseWorksSkippedOutcomeMock = (
  overrideResponse: Partial<WorksSkippedOutcome> = {},
): WorksSkippedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["skipped"] as const),
    reason: faker.helpers.arrayElement([
      "already",
      "work_has_files",
      "work_gone_upstream",
      "work_not_found",
    ] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksMonitoredResponseWorksFailedOutcomeMock = (
  overrideResponse: Partial<WorksFailedOutcome> = {},
): WorksFailedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksMonitoredResponseMock = (
  overrideResponse: Partial<Extract<WorksOutcomeList, object>> = {},
): WorksOutcomeList => ({
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getSetWorksMonitoredResponseWorksDoneOutcomeMock() },
      { ...getSetWorksMonitoredResponseWorksSkippedOutcomeMock() },
      { ...getSetWorksMonitoredResponseWorksFailedOutcomeMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getSetWorksMonitoringResponseWorksMonitoringDoneOutcomeMock = (
  overrideResponse: Partial<WorksMonitoringDoneOutcome> = {},
): WorksMonitoringDoneOutcome => ({
  ...{
    counts: faker.helpers.arrayElement([
      {
        asides: faker.number.int(),
        choices: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            monitors: faker.number.int(),
            monitorsNew: faker.datatype.boolean(),
            preset: faker.helpers.arrayElement([
              "all",
              "future",
              "missing",
              "existing",
              "first",
              "latest",
              "none",
            ] as const),
          }),
        ),
      },
      null,
    ]),
    outcome: faker.helpers.arrayElement(["done"] as const),
    start: faker.number.int(),
    stop: faker.number.int(),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksMonitoringResponseWorksMonitoringSkippedOutcomeMock = (
  overrideResponse: Partial<WorksMonitoringSkippedOutcome> = {},
): WorksMonitoringSkippedOutcome => ({
  ...{
    counts: faker.helpers.arrayElement([
      {
        asides: faker.number.int(),
        choices: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => ({
            monitors: faker.number.int(),
            monitorsNew: faker.datatype.boolean(),
            preset: faker.helpers.arrayElement([
              "all",
              "future",
              "missing",
              "existing",
              "first",
              "latest",
              "none",
            ] as const),
          }),
        ),
      },
      null,
    ]),
    outcome: faker.helpers.arrayElement(["skipped"] as const),
    reason: faker.helpers.arrayElement(["already"] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksMonitoringResponseWorksFailedOutcomeMock = (
  overrideResponse: Partial<WorksFailedOutcome> = {},
): WorksFailedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksMonitoringResponseMock = (
  overrideResponse: Partial<Extract<WorksMonitoringList, object>> = {},
): WorksMonitoringList => ({
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getSetWorksMonitoringResponseWorksMonitoringDoneOutcomeMock() },
      { ...getSetWorksMonitoringResponseWorksMonitoringSkippedOutcomeMock() },
      { ...getSetWorksMonitoringResponseWorksFailedOutcomeMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getSetWorksProfileResponseWorksDoneOutcomeMock = (
  overrideResponse: Partial<WorksDoneOutcome> = {},
): WorksDoneOutcome => ({
  ...{ outcome: faker.helpers.arrayElement(["done"] as const), workId: faker.string.uuid() },
  ...overrideResponse,
});

export const getSetWorksProfileResponseWorksSkippedOutcomeMock = (
  overrideResponse: Partial<WorksSkippedOutcome> = {},
): WorksSkippedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["skipped"] as const),
    reason: faker.helpers.arrayElement([
      "already",
      "work_has_files",
      "work_gone_upstream",
      "work_not_found",
    ] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksProfileResponseWorksFailedOutcomeMock = (
  overrideResponse: Partial<WorksFailedOutcome> = {},
): WorksFailedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksProfileResponseMock = (
  overrideResponse: Partial<Extract<WorksProfileChanged, object>> = {},
): WorksProfileChanged => ({
  impact: {
    downloading: faker.number.int({ min: 0 }),
    newlyWanted: faker.number.int({ min: 0 }),
    ownList: faker.number.int({ min: 0 }),
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
    unwanted: { bytes: faker.number.int({ min: 0 }), files: faker.number.int({ min: 0 }) },
    works: faker.number.int({ min: 0 }),
  },
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getSetWorksProfileResponseWorksDoneOutcomeMock() },
      { ...getSetWorksProfileResponseWorksSkippedOutcomeMock() },
      { ...getSetWorksProfileResponseWorksFailedOutcomeMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getRefreshWorksResponseWorksDoneOutcomeMock = (
  overrideResponse: Partial<WorksDoneOutcome> = {},
): WorksDoneOutcome => ({
  ...{ outcome: faker.helpers.arrayElement(["done"] as const), workId: faker.string.uuid() },
  ...overrideResponse,
});

export const getRefreshWorksResponseWorksSkippedOutcomeMock = (
  overrideResponse: Partial<WorksSkippedOutcome> = {},
): WorksSkippedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["skipped"] as const),
    reason: faker.helpers.arrayElement([
      "already",
      "work_has_files",
      "work_gone_upstream",
      "work_not_found",
    ] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getRefreshWorksResponseWorksFailedOutcomeMock = (
  overrideResponse: Partial<WorksFailedOutcome> = {},
): WorksFailedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getRefreshWorksResponseMock = (
  overrideResponse: Partial<Extract<WorksOutcomeList, object>> = {},
): WorksOutcomeList => ({
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getRefreshWorksResponseWorksDoneOutcomeMock() },
      { ...getRefreshWorksResponseWorksSkippedOutcomeMock() },
      { ...getRefreshWorksResponseWorksFailedOutcomeMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getSetWorksRootResponseWorksDoneOutcomeMock = (
  overrideResponse: Partial<WorksDoneOutcome> = {},
): WorksDoneOutcome => ({
  ...{ outcome: faker.helpers.arrayElement(["done"] as const), workId: faker.string.uuid() },
  ...overrideResponse,
});

export const getSetWorksRootResponseWorksSkippedOutcomeMock = (
  overrideResponse: Partial<WorksSkippedOutcome> = {},
): WorksSkippedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["skipped"] as const),
    reason: faker.helpers.arrayElement([
      "already",
      "work_has_files",
      "work_gone_upstream",
      "work_not_found",
    ] as const),
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksRootResponseWorksFailedOutcomeMock = (
  overrideResponse: Partial<WorksFailedOutcome> = {},
): WorksFailedOutcome => ({
  ...{
    outcome: faker.helpers.arrayElement(["failed"] as const),
    problem: {
      code: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { detail: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      title: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    workId: faker.string.uuid(),
  },
  ...overrideResponse,
});

export const getSetWorksRootResponseMock = (
  overrideResponse: Partial<Extract<WorksOutcomeList, object>> = {},
): WorksOutcomeList => ({
  outcomes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getSetWorksRootResponseWorksDoneOutcomeMock() },
      { ...getSetWorksRootResponseWorksSkippedOutcomeMock() },
      { ...getSetWorksRootResponseWorksFailedOutcomeMock() },
    ]),
  ),
  ...overrideResponse,
});

export const getSearchWorksResponseMock = (
  overrideResponse: Partial<Extract<WorksSearchResult, object>> = {},
): WorksSearchResult => ({
  rows: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    addedBy: faker.helpers.arrayElement(["admin", "scan"] as const),
    availableOn: faker.helpers.arrayElement([
      faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
      null,
    ]),
    bytes: faker.number.int({ min: 0 }),
    cells: {
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
    files: faker.number.int({ min: 0 }),
    goneFiles: faker.number.int({ min: 0 }),
    goneUpstreamAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    id: faker.string.uuid(),
    itemStatus: {
      ...(faker.datatype.boolean() ? { downloading: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { failed: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { met: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { missing: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { unreleased: faker.number.int({ min: 1 }) } : {}),
      ...(faker.datatype.boolean() ? { upgradable: faker.number.int({ min: 1 }) } : {}),
    },
    items: faker.number.int({ min: 0 }),
    kept: faker.number.int({ min: 0 }),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    monitored: faker.helpers.arrayElement(["all", "some", "none"] as const),
    needsYou: faker.number.int({ min: 0 }),
    posterUrl: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    profileId: faker.string.uuid(),
    rating: faker.helpers.arrayElement([
      {
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
      },
      null,
    ]),
    refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
    rootId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
    status: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "failed",
        "missing",
        "downloading",
        "upgradable",
        "unreleased",
        "met",
      ] as const),
      null,
    ]),
    values: {
      [faker.string.alphanumeric(5)]: faker.helpers.arrayElement([
        faker.number.float({ fractionDigits: 2 }),
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      ]),
    },
  })),
  workIds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getGetWorkResponseWorksTextValueMock = (
  overrideResponse: Partial<WorksTextValue> = {},
): WorksTextValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["text"] as const),
    text: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksNumberValueMock = (
  overrideResponse: Partial<WorksNumberValue> = {},
): WorksNumberValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["number"] as const),
    number: faker.number.float({ fractionDigits: 2 }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksDateValueMock = (
  overrideResponse: Partial<WorksDateValue> = {},
): WorksDateValue => ({
  ...{
    date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["date"] as const),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksDurationValueMock = (
  overrideResponse: Partial<WorksDurationValue> = {},
): WorksDurationValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["duration"] as const),
    ms: faker.number.int({ min: 0 }),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksTextsValueMock = (
  overrideResponse: Partial<WorksTextsValue> = {},
): WorksTextsValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["texts"] as const),
    texts: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.string.alpha({ length: { min: 10, max: 20 } }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksPeopleValueMock = (
  overrideResponse: Partial<WorksPeopleValue> = {},
): WorksPeopleValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["people"] as const),
    people: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        department: faker.string.alpha({ length: { min: 10, max: 20 } }),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        photoUrl: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        role: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
  },
  ...overrideResponse,
});

export const getGetWorkResponseWorksLinkValueMock = (
  overrideResponse: Partial<WorksLinkValue> = {},
): WorksLinkValue => ({
  ...{
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["link"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  },
  ...overrideResponse,
});

export const getGetWorkResponseSettingDurationFieldMock = (
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

export const getGetWorkResponseSettingCountFieldMock = (
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

export const getGetWorkResponseSettingChoiceFieldMock = (
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

export const getGetWorkResponseSettingWindowFieldMock = (
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

export const getGetWorkResponseSettingPortFieldMock = (
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

export const getGetWorkResponseSettingSwitchFieldMock = (
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

export const getGetWorkResponseSettingRateFieldMock = (
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

export const getGetWorkResponseSettingAddressFieldMock = (
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

export const getGetWorkResponseSettingFolderFieldMock = (
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

export const getGetWorkResponseSettingSpaceFieldMock = (
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

export const getGetWorkResponseSettingTemplateFieldMock = (
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

export const getGetWorkResponseSettingListFieldMock = (
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

export const getGetWorkResponseSettingSizeFieldMock = (
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

export const getGetWorkResponseMock = (
  overrideResponse: Partial<Extract<WorksWorkView, object>> = {},
): WorksWorkView => ({
  addedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
      url: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  contentType: faker.helpers.fromRegExp("^[a-z][a-z0-9]{1,15}$"),
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkResponseWorksTextValueMock() },
      { ...getGetWorkResponseWorksNumberValueMock() },
      { ...getGetWorkResponseWorksDateValueMock() },
      { ...getGetWorkResponseWorksDurationValueMock() },
      { ...getGetWorkResponseWorksTextsValueMock() },
      { ...getGetWorkResponseWorksPeopleValueMock() },
      { ...getGetWorkResponseWorksLinkValueMock() },
    ]),
  ),
  goneUpstreamAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  id: faker.string.uuid(),
  items: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
        url: faker.string.alpha({ length: { min: 10, max: 20 } }),
      }),
    ),
    aside: faker.datatype.boolean(),
    availability: faker.helpers.arrayElement([
      {
        basis: faker.string.alpha({ length: { min: 10, max: 20 } }),
        date: faker.helpers.fromRegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"),
      },
      null,
    ]),
    fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.helpers.arrayElement([
        { ...getGetWorkResponseWorksTextValueMock() },
        { ...getGetWorkResponseWorksNumberValueMock() },
        { ...getGetWorkResponseWorksDateValueMock() },
        { ...getGetWorkResponseWorksDurationValueMock() },
        { ...getGetWorkResponseWorksTextsValueMock() },
        { ...getGetWorkResponseWorksPeopleValueMock() },
        { ...getGetWorkResponseWorksLinkValueMock() },
      ]),
    ),
    id: faker.string.uuid(),
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    levelId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    monitored: faker.datatype.boolean(),
    position: faker.number.int({ min: 1 }),
    ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => ({
        fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
        label: faker.string.alpha({ length: { min: 10, max: 20 } }),
        scale: faker.number.float({ min: 0, fractionDigits: 2 }),
        source: faker.string.alpha({ length: { min: 10, max: 20 } }),
        value: faker.number.float({ min: 0, fractionDigits: 2 }),
        votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
      }),
    ),
    withdrawnAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
  })),
  key: faker.string.alpha({ length: { min: 10, max: 20 } }),
  label: faker.string.alpha({ length: { min: 10, max: 20 } }),
  levelFolders: faker.datatype.boolean(),
  levels: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      artwork: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
          url: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      aside: faker.datatype.boolean(),
      depth: faker.number.int({ min: 1 }),
      fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () =>
          faker.helpers.arrayElement([
            { ...getGetWorkResponseWorksTextValueMock() },
            { ...getGetWorkResponseWorksNumberValueMock() },
            { ...getGetWorkResponseWorksDateValueMock() },
            { ...getGetWorkResponseWorksDurationValueMock() },
            { ...getGetWorkResponseWorksTextsValueMock() },
            { ...getGetWorkResponseWorksPeopleValueMock() },
            { ...getGetWorkResponseWorksLinkValueMock() },
          ]),
      ),
      id: faker.string.uuid(),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      parentId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      position: faker.number.int({ min: 1 }),
      withdrawnAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
    }),
  ),
  links: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    key: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.helpers.arrayElement(["page", "trailer"] as const),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    url: faker.internet.url(),
  })),
  monitorAsides: faker.datatype.boolean(),
  monitorNew: faker.datatype.boolean(),
  openIn: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      kind: faker.helpers.arrayElement(["jellyfin", "emby"] as const),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      serverId: faker.string.uuid(),
      url: faker.internet.url(),
    }),
  ),
  options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getGetWorkResponseSettingDurationFieldMock() },
      { ...getGetWorkResponseSettingCountFieldMock() },
      { ...getGetWorkResponseSettingChoiceFieldMock() },
      { ...getGetWorkResponseSettingWindowFieldMock() },
      { ...getGetWorkResponseSettingPortFieldMock() },
      { ...getGetWorkResponseSettingSwitchFieldMock() },
      { ...getGetWorkResponseSettingRateFieldMock() },
      { ...getGetWorkResponseSettingAddressFieldMock() },
      { ...getGetWorkResponseSettingFolderFieldMock() },
      { ...getGetWorkResponseSettingSpaceFieldMock() },
      { ...getGetWorkResponseSettingTemplateFieldMock() },
      { ...getGetWorkResponseSettingListFieldMock() },
      { ...getGetWorkResponseSettingSizeFieldMock() },
    ]),
  ),
  profile: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
  ratings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      scale: faker.number.float({ min: 0, fractionDigits: 2 }),
      source: faker.string.alpha({ length: { min: 10, max: 20 } }),
      value: faker.number.float({ min: 0, fractionDigits: 2 }),
      votes: faker.helpers.arrayElement([faker.number.int({ min: 0 }), null]),
    }),
  ),
  refreshedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  root: faker.helpers.arrayElement([
    {
      id: faker.string.uuid(),
      nodeId: faker.string.uuid(),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
    },
    null,
  ]),
  sortLabel: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getSetWorkOptionsResponseSettingDurationFieldMock = (
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

export const getSetWorkOptionsResponseSettingCountFieldMock = (
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

export const getSetWorkOptionsResponseSettingChoiceFieldMock = (
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

export const getSetWorkOptionsResponseSettingWindowFieldMock = (
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

export const getSetWorkOptionsResponseSettingPortFieldMock = (
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

export const getSetWorkOptionsResponseSettingSwitchFieldMock = (
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

export const getSetWorkOptionsResponseSettingRateFieldMock = (
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

export const getSetWorkOptionsResponseSettingAddressFieldMock = (
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

export const getSetWorkOptionsResponseSettingFolderFieldMock = (
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

export const getSetWorkOptionsResponseSettingSpaceFieldMock = (
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

export const getSetWorkOptionsResponseSettingTemplateFieldMock = (
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

export const getSetWorkOptionsResponseSettingListFieldMock = (
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

export const getSetWorkOptionsResponseSettingSizeFieldMock = (
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

export const getSetWorkOptionsResponseMock = (
  overrideResponse: Partial<Extract<WorksOptionsChanged, object>> = {},
): WorksOptionsChanged => ({
  levelFolders: faker.datatype.boolean(),
  moves: faker.datatype.boolean(),
  options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.helpers.arrayElement([
      { ...getSetWorkOptionsResponseSettingDurationFieldMock() },
      { ...getSetWorkOptionsResponseSettingCountFieldMock() },
      { ...getSetWorkOptionsResponseSettingChoiceFieldMock() },
      { ...getSetWorkOptionsResponseSettingWindowFieldMock() },
      { ...getSetWorkOptionsResponseSettingPortFieldMock() },
      { ...getSetWorkOptionsResponseSettingSwitchFieldMock() },
      { ...getSetWorkOptionsResponseSettingRateFieldMock() },
      { ...getSetWorkOptionsResponseSettingAddressFieldMock() },
      { ...getSetWorkOptionsResponseSettingFolderFieldMock() },
      { ...getSetWorkOptionsResponseSettingSpaceFieldMock() },
      { ...getSetWorkOptionsResponseSettingTemplateFieldMock() },
      { ...getSetWorkOptionsResponseSettingListFieldMock() },
      { ...getSetWorkOptionsResponseSettingSizeFieldMock() },
    ]),
  ),
  withdraws: faker.number.int(),
  ...overrideResponse,
});

export const getListContentTypesMockHandler = (
  overrideResponse?:
    | WorksContentTypeList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksContentTypeList> | WorksContentTypeList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListContentTypesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSearchCatalogMockHandler = (
  overrideResponse?:
    | WorksCandidateList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksCandidateList> | WorksCandidateList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types/:contentType/catalog",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSearchCatalogResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getPreviewCatalogRecordMockHandler = (
  overrideResponse?:
    | WorksPreviewed
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksPreviewed> | WorksPreviewed),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types/:contentType/catalog/:key",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getPreviewCatalogRecordResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetContentTypeEnabledMockHandler = (
  overrideResponse?:
    | WorksContentType
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksContentType> | WorksContentType),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/content-types/:contentType/enabled",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetContentTypeEnabledResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetContentTypeSettingsMockHandler = (
  overrideResponse?:
    | WorksTypeSettings
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksTypeSettings> | WorksTypeSettings),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/content-types/:contentType/settings",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetContentTypeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateContentTypeSettingsMockHandler = (
  overrideResponse?:
    | WorksTypeSettings
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<WorksTypeSettings> | WorksTypeSettings),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/content-types/:contentType/settings",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateContentTypeSettingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListLibraryMockHandler = (
  overrideResponse?:
    | WorksLibraryRows
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksLibraryRows> | WorksLibraryRows),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/library",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListLibraryResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddWorkMockHandler = (
  overrideResponse?:
    | WorksAdded
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<WorksAdded> | WorksAdded),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteWorksMockHandler = (
  overrideResponse?:
    | WorksDeleted
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<WorksDeleted> | WorksDeleted),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/delete",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getDeleteWorksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetWorksMonitoredMockHandler = (
  overrideResponse?:
    | WorksOutcomeList
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksOutcomeList> | WorksOutcomeList),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/monitored",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorksMonitoredResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetWorksMonitoringMockHandler = (
  overrideResponse?:
    | WorksMonitoringList
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksMonitoringList> | WorksMonitoringList),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/monitoring",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorksMonitoringResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetWorksProfileMockHandler = (
  overrideResponse?:
    | WorksProfileChanged
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksProfileChanged> | WorksProfileChanged),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/profile",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorksProfileResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRefreshWorksMockHandler = (
  overrideResponse?:
    | WorksOutcomeList
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<WorksOutcomeList> | WorksOutcomeList),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/works/refresh",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getRefreshWorksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetWorksRootMockHandler = (
  overrideResponse?:
    | WorksOutcomeList
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksOutcomeList> | WorksOutcomeList),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/root",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorksRootResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSearchWorksMockHandler = (
  overrideResponse?:
    | WorksSearchResult
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksSearchResult> | WorksSearchResult),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/search",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSearchWorksResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getGetWorkMockHandler = (
  overrideResponse?:
    | WorksWorkView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<WorksWorkView> | WorksWorkView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/works/:workId",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetWorkResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetWorkMonitoredMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.put>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/:workId/monitored",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getSetWorkOptionsMockHandler = (
  overrideResponse?:
    | WorksOptionsChanged
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<WorksOptionsChanged> | WorksOptionsChanged),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/works/:workId/options",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetWorkOptionsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getWorksMock = () => [
  getListContentTypesMockHandler(),
  getSearchCatalogMockHandler(),
  getPreviewCatalogRecordMockHandler(),
  getSetContentTypeEnabledMockHandler(),
  getGetContentTypeSettingsMockHandler(),
  getUpdateContentTypeSettingsMockHandler(),
  getListLibraryMockHandler(),
  getAddWorkMockHandler(),
  getDeleteWorksMockHandler(),
  getSetWorksMonitoredMockHandler(),
  getSetWorksMonitoringMockHandler(),
  getSetWorksProfileMockHandler(),
  getRefreshWorksMockHandler(),
  getSetWorksRootMockHandler(),
  getSearchWorksMockHandler(),
  getGetWorkMockHandler(),
  getSetWorkMonitoredMockHandler(),
  getSetWorkOptionsMockHandler(),
];
