import { faker } from "@faker-js/faker";

import { HttpResponse, http } from "msw";
import type { RequestHandlerOptions } from "msw";

import type {
  HoldersOutputBody,
  KindsOutputBody,
  MappingsOutputBody,
  MediaserversAccountView,
  MediaserversChanges,
  MediaserversLibraries,
  MediaserversPlexPin,
  MediaserversPlexPinState,
  MediaserversServerView,
  MediaserversSessions,
  MediaserversSignedIn,
  MediaserversTested,
  PlexServersOutputBody,
  ServersOutputBody,
} from "../../model";

export const getListMediaServersResponseMock = (
  overrideResponse: Partial<Extract<ServersOutputBody, object>> = {},
): ServersOutputBody => ({
  servers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      address: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { browserAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      ...(faker.datatype.boolean()
        ? { checkedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      contact: faker.helpers.arrayElement([
        "unknown",
        "responding",
        "not_responding",
        "sign_in_refused",
        "other_server",
      ] as const),
      ...(faker.datatype.boolean()
        ? { contactSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      fallback: faker.helpers.arrayElement(["available", "none", "untested"] as const),
      ...(faker.datatype.boolean()
        ? { fallbackAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      id: faker.string.uuid(),
      identity: faker.string.alpha({ length: { min: 10, max: 20 } }),
      kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { lastContactAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      ...(faker.datatype.boolean()
        ? { lastSendAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      mappingsNeeded: faker.number.int({ min: 0 }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean() ? { plexPass: faker.datatype.boolean() } : {}),
      reachingNode: {
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      ...(faker.datatype.boolean()
        ? {
            runsOnNode: {
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      sends: faker.helpers.arrayElement(["ok", "failing"] as const),
      state: {
        level: faker.helpers.arrayElement(["ok", "warn", "fail"] as const),
        words: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      ...(faker.datatype.boolean()
        ? {
            unhealthy: {
              reason: faker.helpers.arrayElement([
                "not_responding",
                "sends_failing",
                "sign_in_refused",
                "other_server",
              ] as const),
              since: faker.date.past().toISOString().slice(0, 19) + "Z",
            },
          }
        : {}),
      version: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getAddMediaServerResponseMock = (
  overrideResponse: Partial<Extract<MediaserversServerView, object>> = {},
): MediaserversServerView => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { browserAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...(faker.datatype.boolean()
    ? { checkedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  contact: faker.helpers.arrayElement([
    "unknown",
    "responding",
    "not_responding",
    "sign_in_refused",
    "other_server",
  ] as const),
  ...(faker.datatype.boolean()
    ? { contactSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  fallback: faker.helpers.arrayElement(["available", "none", "untested"] as const),
  ...(faker.datatype.boolean()
    ? { fallbackAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  id: faker.string.uuid(),
  identity: faker.string.alpha({ length: { min: 10, max: 20 } }),
  kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { lastContactAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...(faker.datatype.boolean()
    ? { lastSendAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  mappingsNeeded: faker.number.int({ min: 0 }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean() ? { plexPass: faker.datatype.boolean() } : {}),
  reachingNode: {
    id: faker.string.uuid(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        runsOnNode: {
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  sends: faker.helpers.arrayElement(["ok", "failing"] as const),
  state: {
    level: faker.helpers.arrayElement(["ok", "warn", "fail"] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        unhealthy: {
          reason: faker.helpers.arrayElement([
            "not_responding",
            "sends_failing",
            "sign_in_refused",
            "other_server",
          ] as const),
          since: faker.date.past().toISOString().slice(0, 19) + "Z",
        },
      }
    : {}),
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListMediaServerKindsResponseMock = (
  overrideResponse: Partial<Extract<KindsOutputBody, object>> = {},
): KindsOutputBody => ({
  kinds: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(() => ({
    browser: faker.datatype.boolean(),
    floor: faker.string.alpha({ length: { min: 10, max: 20 } }),
    kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
    label: faker.string.alpha({ length: { min: 10, max: 20 } }),
    signIn: faker.helpers.arrayElement(["plex_pin", "api_key_or_admin"] as const),
  })),
  ...overrideResponse,
});

export const getGetPlexAccountResponseMock = (
  overrideResponse: Partial<Extract<MediaserversAccountView, object>> = {},
): MediaserversAccountView => ({
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  plexPass: faker.datatype.boolean(),
  refused: faker.datatype.boolean(),
  ...(faker.datatype.boolean()
    ? { renewedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  signedInAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  ...overrideResponse,
});

export const getStartPlexSignInResponseMock = (
  overrideResponse: Partial<Extract<MediaserversPlexPin, object>> = {},
): MediaserversPlexPin => ({
  authUrl: faker.internet.url(),
  expiresAt: faker.date.past().toISOString().slice(0, 19) + "Z",
  pinId: faker.number.int(),
  ...overrideResponse,
});

export const getGetPlexSignInResponseMock = (
  overrideResponse: Partial<Extract<MediaserversPlexPinState, object>> = {},
): MediaserversPlexPinState => ({
  state: faker.helpers.arrayElement(["waiting", "expired", "signed_in"] as const),
  ...overrideResponse,
});

export const getListPlexServersResponseMock = (
  overrideResponse: Partial<Extract<PlexServersOutputBody, object>> = {},
): PlexServersOutputBody => ({
  servers: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      connections: Array.from(
        { length: faker.number.int({ min: 1, max: 10 }) },
        (_, i) => i + 1,
      ).map(() => ({
        https: faker.datatype.boolean(),
        local: faker.datatype.boolean(),
        reachedFrom: Array.from(
          { length: faker.number.int({ min: 1, max: 10 }) },
          (_, i) => i + 1,
        ).map(() => ({ nodeId: faker.string.uuid(), ok: faker.datatype.boolean() })),
        uri: faker.internet.url(),
      })),
      machineIdentifier: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      suggestedReaching: {
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      ...(faker.datatype.boolean()
        ? {
            suggestedRunsOn: {
              id: faker.string.uuid(),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      version: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...overrideResponse,
});

export const getSignInToMediaServerResponseMock = (
  overrideResponse: Partial<Extract<MediaserversSignedIn, object>> = {},
): MediaserversSignedIn => ({
  apiKey: faker.string.alpha({ length: { min: 10, max: 20 } }),
  deviceId: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getGetMediaServerResponseMock = (
  overrideResponse: Partial<Extract<MediaserversServerView, object>> = {},
): MediaserversServerView => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { browserAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...(faker.datatype.boolean()
    ? { checkedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  contact: faker.helpers.arrayElement([
    "unknown",
    "responding",
    "not_responding",
    "sign_in_refused",
    "other_server",
  ] as const),
  ...(faker.datatype.boolean()
    ? { contactSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  fallback: faker.helpers.arrayElement(["available", "none", "untested"] as const),
  ...(faker.datatype.boolean()
    ? { fallbackAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  id: faker.string.uuid(),
  identity: faker.string.alpha({ length: { min: 10, max: 20 } }),
  kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { lastContactAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...(faker.datatype.boolean()
    ? { lastSendAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  mappingsNeeded: faker.number.int({ min: 0 }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean() ? { plexPass: faker.datatype.boolean() } : {}),
  reachingNode: {
    id: faker.string.uuid(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        runsOnNode: {
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  sends: faker.helpers.arrayElement(["ok", "failing"] as const),
  state: {
    level: faker.helpers.arrayElement(["ok", "warn", "fail"] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        unhealthy: {
          reason: faker.helpers.arrayElement([
            "not_responding",
            "sends_failing",
            "sign_in_refused",
            "other_server",
          ] as const),
          since: faker.date.past().toISOString().slice(0, 19) + "Z",
        },
      }
    : {}),
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getUpdateMediaServerResponseMock = (
  overrideResponse: Partial<Extract<MediaserversServerView, object>> = {},
): MediaserversServerView => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { browserAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...(faker.datatype.boolean()
    ? { checkedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  contact: faker.helpers.arrayElement([
    "unknown",
    "responding",
    "not_responding",
    "sign_in_refused",
    "other_server",
  ] as const),
  ...(faker.datatype.boolean()
    ? { contactSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  fallback: faker.helpers.arrayElement(["available", "none", "untested"] as const),
  ...(faker.datatype.boolean()
    ? { fallbackAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  id: faker.string.uuid(),
  identity: faker.string.alpha({ length: { min: 10, max: 20 } }),
  kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { lastContactAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...(faker.datatype.boolean()
    ? { lastSendAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  mappingsNeeded: faker.number.int({ min: 0 }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean() ? { plexPass: faker.datatype.boolean() } : {}),
  reachingNode: {
    id: faker.string.uuid(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        runsOnNode: {
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  sends: faker.helpers.arrayElement(["ok", "failing"] as const),
  state: {
    level: faker.helpers.arrayElement(["ok", "warn", "fail"] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        unhealthy: {
          reason: faker.helpers.arrayElement([
            "not_responding",
            "sends_failing",
            "sign_in_refused",
            "other_server",
          ] as const),
          since: faker.date.past().toISOString().slice(0, 19) + "Z",
        },
      }
    : {}),
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListMediaServerChangesResponseMock = (
  overrideResponse: Partial<Extract<MediaserversChanges, object>> = {},
): MediaserversChanges => ({
  changes: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      attempts: faker.number.int({ min: 0 }),
      createdAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      ...(faker.datatype.boolean()
        ? { fromPath: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      id: faker.string.uuid(),
      kind: faker.helpers.arrayElement([
        "added",
        "replaced",
        "removed",
        "moved",
        "sidecar_added",
        "sidecar_replaced",
        "sidecar_removed",
        "library",
      ] as const),
      lost: faker.helpers.arrayElements([
        "watch_state",
        "play_count",
        "last_played",
        "rating",
        "collections",
      ] as const),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean()
        ? { result: faker.helpers.arrayElement(["accepted", "covered", "converted"] as const) }
        : {}),
      ...(faker.datatype.boolean()
        ? { sentAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
        : {}),
      state: faker.helpers.arrayElement(["prepared", "due", "sending", "sent"] as const),
      step: faker.helpers.arrayElement([
        "none",
        "snapshotted",
        "scanned",
        "source_scanned",
        "confirmed",
        "idle",
        "reported",
        "settled",
        "old_removed",
        "new_added",
        "new_found",
        "restored",
      ] as const),
      words: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  ...(faker.datatype.boolean()
    ? { next: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...overrideResponse,
});

export const getConfirmMediaServerIdentityResponseMock = (
  overrideResponse: Partial<Extract<MediaserversServerView, object>> = {},
): MediaserversServerView => ({
  address: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { browserAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  ...(faker.datatype.boolean()
    ? { checkedAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  contact: faker.helpers.arrayElement([
    "unknown",
    "responding",
    "not_responding",
    "sign_in_refused",
    "other_server",
  ] as const),
  ...(faker.datatype.boolean()
    ? { contactSince: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  fallback: faker.helpers.arrayElement(["available", "none", "untested"] as const),
  ...(faker.datatype.boolean()
    ? { fallbackAddress: faker.string.alpha({ length: { min: 10, max: 20 } }) }
    : {}),
  id: faker.string.uuid(),
  identity: faker.string.alpha({ length: { min: 10, max: 20 } }),
  kind: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean()
    ? { lastContactAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  ...(faker.datatype.boolean()
    ? { lastSendAt: faker.date.past().toISOString().slice(0, 19) + "Z" }
    : {}),
  mappingsNeeded: faker.number.int({ min: 0 }),
  name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...(faker.datatype.boolean() ? { plexPass: faker.datatype.boolean() } : {}),
  reachingNode: {
    id: faker.string.uuid(),
    name: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        runsOnNode: {
          id: faker.string.uuid(),
          name: faker.string.alpha({ length: { min: 10, max: 20 } }),
        },
      }
    : {}),
  sends: faker.helpers.arrayElement(["ok", "failing"] as const),
  state: {
    level: faker.helpers.arrayElement(["ok", "warn", "fail"] as const),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...(faker.datatype.boolean()
    ? {
        unhealthy: {
          reason: faker.helpers.arrayElement([
            "not_responding",
            "sends_failing",
            "sign_in_refused",
            "other_server",
          ] as const),
          since: faker.date.past().toISOString().slice(0, 19) + "Z",
        },
      }
    : {}),
  version: faker.string.alpha({ length: { min: 10, max: 20 } }),
  ...overrideResponse,
});

export const getListMediaServerCredentialHoldersResponseMock = (
  overrideResponse: Partial<Extract<HoldersOutputBody, object>> = {},
): HoldersOutputBody => ({
  holders: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      current: faker.datatype.boolean(),
      fetchedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      node: { id: faker.string.uuid(), name: faker.string.alpha({ length: { min: 10, max: 20 } }) },
    }),
  ),
  ...overrideResponse,
});

export const getListMediaServerLibrariesResponseMock = (
  overrideResponse: Partial<Extract<MediaserversLibraries, object>> = {},
): MediaserversLibraries => ({
  libraries: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      folders: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      key: faker.string.alpha({ length: { min: 10, max: 20 } }),
      name: faker.string.alpha({ length: { min: 10, max: 20 } }),
    }),
  ),
  mappings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      confirmed: faker.datatype.boolean(),
      guess: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? {
            library: {
              folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      rootId: faker.string.uuid(),
      rootName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootNode: {
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      scanned: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { serverFolder: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      shown: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { words: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    }),
  ),
  ...overrideResponse,
});

export const getSaveMediaServerMappingsResponseMock = (
  overrideResponse: Partial<Extract<MappingsOutputBody, object>> = {},
): MappingsOutputBody => ({
  mappings: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      confirmed: faker.datatype.boolean(),
      guess: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? {
            library: {
              folder: faker.string.alpha({ length: { min: 10, max: 20 } }),
              key: faker.string.alpha({ length: { min: 10, max: 20 } }),
              name: faker.string.alpha({ length: { min: 10, max: 20 } }),
            },
          }
        : {}),
      rootId: faker.string.uuid(),
      rootName: faker.string.alpha({ length: { min: 10, max: 20 } }),
      rootNode: {
        id: faker.string.uuid(),
        name: faker.string.alpha({ length: { min: 10, max: 20 } }),
      },
      scanned: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { serverFolder: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
      shown: faker.datatype.boolean(),
      ...(faker.datatype.boolean()
        ? { words: faker.string.alpha({ length: { min: 10, max: 20 } }) }
        : {}),
    }),
  ),
  ...overrideResponse,
});

export const getListMediaServerSessionsResponseMock = (
  overrideResponse: Partial<Extract<MediaserversSessions, object>> = {},
): MediaserversSessions => ({
  reachable: faker.datatype.boolean(),
  sessions: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
    () => ({
      decision: faker.helpers.arrayElement(["direct_play", "copy", "transcode"] as const),
      hardware: faker.datatype.boolean(),
      path: faker.string.alpha({ length: { min: 10, max: 20 } }),
      playId: faker.string.uuid(),
      player: faker.string.alpha({ length: { min: 10, max: 20 } }),
      product: faker.string.alpha({ length: { min: 10, max: 20 } }),
      reasons: Array.from({ length: faker.number.int({ min: 1, max: 10 }) }, (_, i) => i + 1).map(
        () => faker.string.alpha({ length: { min: 10, max: 20 } }),
      ),
      startedAt: faker.date.past().toISOString().slice(0, 19) + "Z",
      user: faker.string.alpha({ length: { min: 10, max: 20 } }),
      ...(faker.datatype.boolean() ? { workId: faker.string.uuid() } : {}),
    }),
  ),
  ...overrideResponse,
});

export const getTestMediaServerResponseMock = (
  overrideResponse: Partial<Extract<MediaserversTested, object>> = {},
): MediaserversTested => ({
  controller: {
    ok: faker.datatype.boolean(),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  fallback: faker.helpers.arrayElement(["available", "none", "untested"] as const),
  reaching: {
    ok: faker.datatype.boolean(),
    words: faker.string.alpha({ length: { min: 10, max: 20 } }),
  },
  ...overrideResponse,
});

export const getListMediaServersMockHandler = (
  overrideResponse?:
    | ServersOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<ServersOutputBody> | ServersOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMediaServersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getAddMediaServerMockHandler = (
  overrideResponse?:
    | MediaserversServerView
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MediaserversServerView> | MediaserversServerView),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/media-servers",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getAddMediaServerResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getListMediaServerKindsMockHandler = (
  overrideResponse?:
    | KindsOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<KindsOutputBody> | KindsOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/kinds",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMediaServerKindsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSignOutOfPlexMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/media-servers/plex/account",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetPlexAccountMockHandler = (
  overrideResponse?:
    | MediaserversAccountView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MediaserversAccountView> | MediaserversAccountView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/plex/account",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetPlexAccountResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getStartPlexSignInMockHandler = (
  overrideResponse?:
    | MediaserversPlexPin
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MediaserversPlexPin> | MediaserversPlexPin),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/media-servers/plex/pin",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getStartPlexSignInResponseMock(),
        { status: 201 },
      );
    },
    options,
  );
};

export const getGetPlexSignInMockHandler = (
  overrideResponse?:
    | MediaserversPlexPinState
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MediaserversPlexPinState> | MediaserversPlexPinState),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/plex/pin/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetPlexSignInResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListPlexServersMockHandler = (
  overrideResponse?:
    | PlexServersOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<PlexServersOutputBody> | PlexServersOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/plex/servers",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListPlexServersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSignInToMediaServerMockHandler = (
  overrideResponse?:
    | MediaserversSignedIn
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MediaserversSignedIn> | MediaserversSignedIn),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/media-servers/sign-in",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSignInToMediaServerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRemoveMediaServerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.delete>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.delete(
    "*/api/v1/media-servers/:id",
    async (info: Parameters<Parameters<typeof http.delete>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 204 });
    },
    options,
  );
};

export const getGetMediaServerMockHandler = (
  overrideResponse?:
    | MediaserversServerView
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MediaserversServerView> | MediaserversServerView),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/:id",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getGetMediaServerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getUpdateMediaServerMockHandler = (
  overrideResponse?:
    | MediaserversServerView
    | ((
        info: Parameters<Parameters<typeof http.patch>[1]>[0],
      ) => Promise<MediaserversServerView> | MediaserversServerView),
  options?: RequestHandlerOptions,
) => {
  return http.patch(
    "*/api/v1/media-servers/:id",
    async (info: Parameters<Parameters<typeof http.patch>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getUpdateMediaServerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListMediaServerChangesMockHandler = (
  overrideResponse?:
    | MediaserversChanges
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MediaserversChanges> | MediaserversChanges),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/:id/changes",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMediaServerChangesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getConfirmMediaServerIdentityMockHandler = (
  overrideResponse?:
    | MediaserversServerView
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MediaserversServerView> | MediaserversServerView),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/media-servers/:id/confirm-identity",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getConfirmMediaServerIdentityResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListMediaServerCredentialHoldersMockHandler = (
  overrideResponse?:
    | HoldersOutputBody
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<HoldersOutputBody> | HoldersOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/:id/credential-holders",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMediaServerCredentialHoldersResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getListMediaServerLibrariesMockHandler = (
  overrideResponse?:
    | MediaserversLibraries
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MediaserversLibraries> | MediaserversLibraries),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/:id/libraries",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMediaServerLibrariesResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getSaveMediaServerMappingsMockHandler = (
  overrideResponse?:
    | MappingsOutputBody
    | ((
        info: Parameters<Parameters<typeof http.put>[1]>[0],
      ) => Promise<MappingsOutputBody> | MappingsOutputBody),
  options?: RequestHandlerOptions,
) => {
  return http.put(
    "*/api/v1/media-servers/:id/mappings",
    async (info: Parameters<Parameters<typeof http.put>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getSaveMediaServerMappingsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getRefreshMediaServerMockHandler = (
  overrideResponse?:
    | void
    | ((info: Parameters<Parameters<typeof http.post>[1]>[0]) => Promise<void> | void),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/media-servers/:id/refresh",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      if (typeof overrideResponse === "function") {
        await overrideResponse(info);
      }

      return new HttpResponse(null, { status: 202 });
    },
    options,
  );
};

export const getListMediaServerSessionsMockHandler = (
  overrideResponse?:
    | MediaserversSessions
    | ((
        info: Parameters<Parameters<typeof http.get>[1]>[0],
      ) => Promise<MediaserversSessions> | MediaserversSessions),
  options?: RequestHandlerOptions,
) => {
  return http.get(
    "*/api/v1/media-servers/:id/sessions",
    async (info: Parameters<Parameters<typeof http.get>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getListMediaServerSessionsResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};

export const getTestMediaServerMockHandler = (
  overrideResponse?:
    | MediaserversTested
    | ((
        info: Parameters<Parameters<typeof http.post>[1]>[0],
      ) => Promise<MediaserversTested> | MediaserversTested),
  options?: RequestHandlerOptions,
) => {
  return http.post(
    "*/api/v1/media-servers/:id/test",
    async (info: Parameters<Parameters<typeof http.post>[1]>[0]) => {
      return HttpResponse.json(
        overrideResponse !== undefined
          ? typeof overrideResponse === "function"
            ? await overrideResponse(info)
            : overrideResponse
          : getTestMediaServerResponseMock(),
        { status: 200 },
      );
    },
    options,
  );
};
export const getMediaserversMock = () => [
  getListMediaServersMockHandler(),
  getAddMediaServerMockHandler(),
  getListMediaServerKindsMockHandler(),
  getSignOutOfPlexMockHandler(),
  getGetPlexAccountMockHandler(),
  getStartPlexSignInMockHandler(),
  getGetPlexSignInMockHandler(),
  getListPlexServersMockHandler(),
  getSignInToMediaServerMockHandler(),
  getRemoveMediaServerMockHandler(),
  getGetMediaServerMockHandler(),
  getUpdateMediaServerMockHandler(),
  getListMediaServerChangesMockHandler(),
  getConfirmMediaServerIdentityMockHandler(),
  getListMediaServerCredentialHoldersMockHandler(),
  getListMediaServerLibrariesMockHandler(),
  getSaveMediaServerMappingsMockHandler(),
  getRefreshMediaServerMockHandler(),
  getListMediaServerSessionsMockHandler(),
  getTestMediaServerMockHandler(),
];
