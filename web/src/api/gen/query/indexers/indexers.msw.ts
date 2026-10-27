import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  IndexersCatalogEntry,
  IndexersDefinitionForm,
  IndexersIndexer,
  IndexersIndexerCatalog,
  IndexersIndexerList,
  IndexersQueryList,
  IndexersReport,
} from "../../model";

export const getGetIndexerCatalogResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerCatalog, object>> = {},
): IndexersIndexerCatalog => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  addressIsDefault: faker.datatype.boolean(),
  entries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      categories: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.number.int()),
      description: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      language: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
      privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
      reason: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      replacedByCustom: faker.datatype.boolean(),
      version: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  nextRefreshAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  notice: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  refreshedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  ...overrideResponse,
});

export const getSetCatalogAddressResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerCatalog, object>> = {},
): IndexersIndexerCatalog => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  addressIsDefault: faker.datatype.boolean(),
  entries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      categories: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => faker.number.int()),
      description: faker.string.alpha({ length: { min: 10, max: 20 } }),
      id: faker.string.alpha({ length: { min: 10, max: 20 } }),
      language: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
      privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
      reason: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      replacedByCustom: faker.datatype.boolean(),
      version: faker.date.past().toISOString().slice(0, 19) + "Z",
    }),
  ),
  nextRefreshAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  notice: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  refreshedAt: faker.helpers.arrayElement([
    faker.date.past().toISOString().slice(0, 19) + "Z",
    null,
  ]),
  ...overrideResponse,
});

export const getAddCustomDefinitionResponseMock = (
  overrideResponse: Partial<Extract<IndexersCatalogEntry, object>> = {},
): IndexersCatalogEntry => ({
  categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => faker.number.int(),
  ),
  description: faker.string.alpha({ length: { min: 10, max: 20 } }),
  id: faker.string.alpha({ length: { min: 10, max: 20 } }),
  language: faker.string.alpha({ length: { min: 10, max: 20 } }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
  privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
  reason: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
  replacedByCustom: faker.datatype.boolean(),
  version: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getGetCatalogDefinitionResponseMock = (
  overrideResponse: Partial<Extract<IndexersDefinitionForm, object>> = {},
): IndexersDefinitionForm => ({
  captcha: faker.datatype.boolean(),
  entry: {
    categories: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
      () => faker.number.int(),
    ),
    description: faker.string.alpha({ length: { min: 10, max: 20 } }),
    id: faker.string.alpha({ length: { min: 10, max: 20 } }),
    language: faker.string.alpha({ length: { min: 10, max: 20 } }),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    origin: faker.helpers.arrayElement(["catalog", "custom"] as const),
    privacy: faker.helpers.arrayElement(["private", "semi-private", "public"] as const),
    reason: faker.helpers.arrayElement([
      faker.string.alpha({ length: { min: 10, max: 20 } }),
      null,
    ]),
    replacedByCustom: faker.datatype.boolean(),
    version: faker.date.past().toISOString().slice(0, 19) + "Z",
  },
  fields: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      default: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      help: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      label: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      options: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => ({
          label: faker.string.alpha({ length: { min: 10, max: 20 } }),
          value: faker.string.alpha({ length: { min: 10, max: 20 } }),
        }),
      ),
      sealed: faker.datatype.boolean(),
      type: faker.helpers.arrayElement([
        "text",
        "password",
        "checkbox",
        "select",
        "info",
        "info_cookie",
        "info_flaresolverr",
        "info_useragent",
        "info_category_8000",
        "cardigannCaptcha",
      ] as const),
    }),
  ),
  links: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  session: faker.datatype.boolean(),
  solver: faker.datatype.boolean(),
  ...overrideResponse,
});

export const getListIndexersResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexerList, object>> = {},
): IndexersIndexerList => ({
  indexers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      allowFailover: faker.datatype.boolean(),
      automaticSearch: faker.datatype.boolean(),
      categories: faker.helpers.arrayElement([
        Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
          faker.number.int(),
        ),
        null,
      ]),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      feedHost: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      grabs: {
        count: faker.number.int({ min: 0 }),
        freeAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        per: faker.helpers.arrayElement(["hour", "day"] as const),
        used: faker.number.int({ min: 0 }),
      },
      homeNodeId: faker.string.uuid(),
      id: faker.string.uuid(),
      kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      on: faker.datatype.boolean(),
      priority: faker.number.int({ min: 1, max: 50 }),
      revision: faker.number.int({ min: 1 }),
      rss: faker.datatype.boolean(),
      searchModes: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => ({
        mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
        params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
          () => faker.string.alpha({ length: { min: 10, max: 20 } }),
        ),
      })),
      searches: {
        count: faker.number.int({ min: 0 }),
        freeAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        per: faker.helpers.arrayElement(["hour", "day"] as const),
        used: faker.number.int({ min: 0 }),
      },
      session: faker.helpers.arrayElement([
        {
          earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
          nodeId: faker.string.uuid(),
          pasted: faker.datatype.boolean(),
        },
        null,
      ]),
      status: {
        cause: faker.helpers.arrayElement([
          faker.helpers.arrayElement([
            "credentials_rejected",
            "captcha_needed",
            "challenge_unsolved",
            "cookie_expired",
            "definition_unloadable",
            "account_refused",
            "request_rejected",
          ] as const),
          null,
        ]),
        lastFailure: faker.helpers.arrayElement([
          {
            at: faker.date.past().toISOString().slice(0, 19) + "Z",
            text: faker.string.alpha({ length: { min: 10, max: 20 } }),
          },
          null,
        ]),
        nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
        rung: faker.number.int({ min: 0 }),
        since: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        solverTried: faker.datatype.boolean(),
        state: faker.helpers.arrayElement([
          "ok",
          "limited",
          "failing",
          "stopped",
          "home_only",
        ] as const),
        successAt: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
        text: faker.helpers.arrayElement([
          faker.string.alpha({ length: { min: 10, max: 20 } }),
          null,
        ]),
        until: faker.helpers.arrayElement([
          faker.date.past().toISOString().slice(0, 19) + "Z",
          null,
        ]),
      },
      waits: faker.helpers.arrayElement([
        {
          cause: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          kind: faker.helpers.arrayElement([
            "found",
            "no_releases",
            "limited",
            "failing",
            "stopped",
            "waiting",
            "late",
            "limit_reached",
            "no_categories",
            "removed",
          ] as const),
          lapsed: faker.datatype.boolean(),
          nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
          releases: faker.number.int({ min: 0 }),
          since: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
          text: faker.helpers.arrayElement([
            faker.string.alpha({ length: { min: 10, max: 20 } }),
            null,
          ]),
          until: faker.helpers.arrayElement([
            faker.date.past().toISOString().slice(0, 19) + "Z",
            null,
          ]),
        },
        null,
      ]),
    }),
  ),
  ...overrideResponse,
});

export const getCreateIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  allowFailover: faker.datatype.boolean(),
  automaticSearch: faker.datatype.boolean(),
  categories: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.number.int(),
    ),
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  feedHost: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  grabs: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  on: faker.datatype.boolean(),
  priority: faker.number.int({ min: 1, max: 50 }),
  revision: faker.number.int({ min: 1 }),
  rss: faker.datatype.boolean(),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  searches: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  session: faker.helpers.arrayElement([
    {
      earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      nodeId: faker.string.uuid(),
      pasted: faker.datatype.boolean(),
    },
    null,
  ]),
  status: {
    cause: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "credentials_rejected",
        "captcha_needed",
        "challenge_unsolved",
        "cookie_expired",
        "definition_unloadable",
        "account_refused",
        "request_rejected",
      ] as const),
      null,
    ]),
    lastFailure: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    rung: faker.number.int({ min: 0 }),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    solverTried: faker.datatype.boolean(),
    state: faker.helpers.arrayElement([
      "ok",
      "limited",
      "failing",
      "stopped",
      "home_only",
    ] as const),
    successAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    text: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  waits: faker.helpers.arrayElement([
    {
      cause: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      kind: faker.helpers.arrayElement([
        "found",
        "no_releases",
        "limited",
        "failing",
        "stopped",
        "waiting",
        "late",
        "limit_reached",
        "no_categories",
        "removed",
      ] as const),
      lapsed: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      releases: faker.number.int({ min: 0 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      text: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getTestIndexerDraftResponseMock = (
  overrideResponse: Partial<Extract<IndexersReport, object>> = {},
): IndexersReport => ({
  items: faker.number.int({ min: 0 }),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  sources: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getGetIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  allowFailover: faker.datatype.boolean(),
  automaticSearch: faker.datatype.boolean(),
  categories: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.number.int(),
    ),
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  feedHost: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  grabs: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  on: faker.datatype.boolean(),
  priority: faker.number.int({ min: 1, max: 50 }),
  revision: faker.number.int({ min: 1 }),
  rss: faker.datatype.boolean(),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  searches: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  session: faker.helpers.arrayElement([
    {
      earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      nodeId: faker.string.uuid(),
      pasted: faker.datatype.boolean(),
    },
    null,
  ]),
  status: {
    cause: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "credentials_rejected",
        "captcha_needed",
        "challenge_unsolved",
        "cookie_expired",
        "definition_unloadable",
        "account_refused",
        "request_rejected",
      ] as const),
      null,
    ]),
    lastFailure: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    rung: faker.number.int({ min: 0 }),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    solverTried: faker.datatype.boolean(),
    state: faker.helpers.arrayElement([
      "ok",
      "limited",
      "failing",
      "stopped",
      "home_only",
    ] as const),
    successAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    text: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  waits: faker.helpers.arrayElement([
    {
      cause: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      kind: faker.helpers.arrayElement([
        "found",
        "no_releases",
        "limited",
        "failing",
        "stopped",
        "waiting",
        "late",
        "limit_reached",
        "no_categories",
        "removed",
      ] as const),
      lapsed: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      releases: faker.number.int({ min: 0 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      text: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getUpdateIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersIndexer, object>> = {},
): IndexersIndexer => ({
  allowFailover: faker.datatype.boolean(),
  automaticSearch: faker.datatype.boolean(),
  categories: faker.helpers.arrayElement([
    Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
      faker.number.int(),
    ),
    null,
  ]),
  createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  feedHost: faker.helpers.arrayElement([
    faker.string.alpha({ length: { min: 10, max: 20 } }),
    null,
  ]),
  grabs: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  homeNodeId: faker.string.uuid(),
  id: faker.string.uuid(),
  kind: faker.helpers.arrayElement(["catalog", "torznab", "rss"] as const),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  on: faker.datatype.boolean(),
  priority: faker.number.int({ min: 1, max: 50 }),
  revision: faker.number.int({ min: 1 }),
  rss: faker.datatype.boolean(),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  searches: {
    count: faker.number.int({ min: 0 }),
    freeAt: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    per: faker.helpers.arrayElement(["hour", "day"] as const),
    used: faker.number.int({ min: 0 }),
  },
  session: faker.helpers.arrayElement([
    {
      earnedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      nodeId: faker.string.uuid(),
      pasted: faker.datatype.boolean(),
    },
    null,
  ]),
  status: {
    cause: faker.helpers.arrayElement([
      faker.helpers.arrayElement([
        "credentials_rejected",
        "captcha_needed",
        "challenge_unsolved",
        "cookie_expired",
        "definition_unloadable",
        "account_refused",
        "request_rejected",
      ] as const),
      null,
    ]),
    lastFailure: faker.helpers.arrayElement([
      {
        at: faker.date.past().toISOString().slice(0, 19) + "Z",
        text: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      null,
    ]),
    nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
    rung: faker.number.int({ min: 0 }),
    since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    solverTried: faker.datatype.boolean(),
    state: faker.helpers.arrayElement([
      "ok",
      "limited",
      "failing",
      "stopped",
      "home_only",
    ] as const),
    successAt: faker.helpers.arrayElement([
      faker.date.past().toISOString().slice(0, 19) + "Z",
      null,
    ]),
    text: faker.helpers.arrayElement([faker.string.alpha({ length: { min: 10, max: 20 } }), null]),
    until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
  },
  waits: faker.helpers.arrayElement([
    {
      cause: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      kind: faker.helpers.arrayElement([
        "found",
        "no_releases",
        "limited",
        "failing",
        "stopped",
        "waiting",
        "late",
        "limit_reached",
        "no_categories",
        "removed",
      ] as const),
      lapsed: faker.datatype.boolean(),
      nodeId: faker.helpers.arrayElement([faker.string.uuid(), null]),
      releases: faker.number.int({ min: 0 }),
      since: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
      text: faker.helpers.arrayElement([
        faker.string.alpha({ length: { min: 10, max: 20 } }),
        null,
      ]),
      until: faker.helpers.arrayElement([faker.date.past().toISOString().slice(0, 19) + "Z", null]),
    },
    null,
  ]),
  ...overrideResponse,
});

export const getListIndexerQueriesResponseMock = (
  overrideResponse: Partial<Extract<IndexersQueryList, object>> = {},
): IndexersQueryList => ({
  nextOffset: faker.helpers.arrayElement([faker.number.int(), null]),
  queries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      completedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      id: faker.string.uuid(),
      outcome: faker.helpers.arrayElement([
        faker.helpers.arrayElement(["ok", "limited", "failing", "stopped", "lost"] as const),
        null,
      ]),
      releases: faker.number.int({ min: 0 }),
      reservedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      startedAt: faker.helpers.arrayElement([
        faker.date.past().toISOString().slice(0, 19) + "Z",
        null,
      ]),
      use: faker.helpers.arrayElement(["search", "recent", "fetch", "test"] as const),
    }),
  ),
  ...overrideResponse,
});

export const getTestIndexerResponseMock = (
  overrideResponse: Partial<Extract<IndexersReport, object>> = {},
): IndexersReport => ({
  items: faker.number.int({ min: 0 }),
  searchModes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      mode: faker.string.alpha({ length: { min: 10, max: 20 } }),
      params: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
    }),
  ),
  sources: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() =>
    faker.string.alpha({ length: { min: 10, max: 20 } }),
  ),
  ...overrideResponse,
});

export const getGetIndexerCatalogMockHandler = (
  overrideResponse?:
    | IndexersIndexerCatalog
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexerCatalog> | IndexersIndexerCatalog),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexer-catalog",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetIndexerCatalogResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSetCatalogAddressMockHandler = (
  overrideResponse?:
    | IndexersIndexerCatalog
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<IndexersIndexerCatalog> | IndexersIndexerCatalog),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/indexer-catalog/address",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSetCatalogAddressResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddCustomDefinitionMockHandler = (
  overrideResponse?:
    | IndexersCatalogEntry
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersCatalogEntry> | IndexersCatalogEntry),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexer-catalog/custom",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddCustomDefinitionResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getDeleteCustomDefinitionMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/indexer-catalog/custom/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetCatalogDefinitionMockHandler = (
  overrideResponse?:
    | IndexersDefinitionForm
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersDefinitionForm> | IndexersDefinitionForm),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexer-catalog/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetCatalogDefinitionResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListIndexersMockHandler = (
  overrideResponse?:
    | IndexersIndexerList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexerList> | IndexersIndexerList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListIndexersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getCreateIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getCreateIndexerResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getTestIndexerDraftMockHandler = (
  overrideResponse?:
    | IndexersReport
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersReport> | IndexersReport),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestIndexerDraftResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getDeleteIndexerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/indexers/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetIndexerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateIndexerMockHandler = (
  overrideResponse?:
    | IndexersIndexer
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<IndexersIndexer> | IndexersIndexer),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/indexers/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateIndexerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListIndexerQueriesMockHandler = (
  overrideResponse?:
    | IndexersQueryList
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<IndexersQueryList> | IndexersQueryList),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/indexers/:id/queries",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListIndexerQueriesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getTestIndexerMockHandler = (
  overrideResponse?:
    | IndexersReport
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<IndexersReport> | IndexersReport),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/indexers/:id/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestIndexerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getIndexersMock = () => [
  getGetIndexerCatalogMockHandler(),
  getSetCatalogAddressMockHandler(),
  getAddCustomDefinitionMockHandler(),
  getDeleteCustomDefinitionMockHandler(),
  getGetCatalogDefinitionMockHandler(),
  getListIndexersMockHandler(),
  getCreateIndexerMockHandler(),
  getTestIndexerDraftMockHandler(),
  getDeleteIndexerMockHandler(),
  getGetIndexerMockHandler(),
  getUpdateIndexerMockHandler(),
  getListIndexerQueriesMockHandler(),
  getTestIndexerMockHandler(),
];
