import { HttpResponse, http } from "msw";

import type {
  MediaserversAccessList,
  MediaserversAccessServer,
  MediaserversAccessView,
  MediaserversActed,
  MediaserversMemberPin,
  MediaserversMemberPinState,
  MediaserversMyAccess,
  MediaserversMyAccessRow,
  MediaserversServerUser,
} from "../api/gen/model";
import { denJellyfin, livingRoomPlex, plexAccount } from "./media-servers";

const api = "/api/v1";

const plexServer: MediaserversAccessServer = {
  id: livingRoomPlex.id,
  name: livingRoomPlex.name,
  kind: "plex",
  access: "plex_tv",
};

const jellyfinServer: MediaserversAccessServer = {
  id: denJellyfin.id,
  name: denJellyfin.name,
  kind: "jellyfin",
  access: "users",
};

const sam = {
  accountId: "0199f3c0-3333-7000-8000-000000000001",
  name: "Sam Rivera",
  email: "sam@example.test",
  disabled: false,
};

const lin = {
  accountId: "0199f3c0-3333-7000-8000-000000000002",
  name: "Lin Park",
  email: "lin@example.test",
  disabled: false,
};

const noor = {
  accountId: "0199f3c0-3333-7000-8000-000000000003",
  name: "Noor Haddad",
  email: "noor@example.test",
  disabled: false,
};

const ravi = {
  accountId: "0199f3c0-3333-7000-8000-000000000004",
  name: "Ravi Shah",
  email: "ravi@example.test",
  disabled: true,
};

const samOnPlex = {
  username: "samr",
  email: "sam.plex@example.test",
  plexHome: false,
  byHand: false,
};

const libraries = ["Films", "Series"];

const changedAt = "2026-11-24T10:00:00Z";

/** Sam's Plex access and Lin's Jellyfin access, one row in each state an Admin reads. */
export const accessRows = {
  offered: {
    id: "0199f3c0-4444-7000-8000-000000000001",
    server: plexServer,
    member: lin,
    state: "offered",
    needsYou: false,
    actions: ["remove"],
    plex: { plexHome: true, byHand: false },
    libraries,
    changedAt,
  },
  waiting: {
    id: "0199f3c0-4444-7000-8000-000000000002",
    server: plexServer,
    member: sam,
    state: "waiting",
    needsYou: true,
    actions: ["shareFriend", "shareHome", "markAdded", "remove"],
    plex: samOnPlex,
    libraries,
    changedAt,
  },
  shared: {
    id: "0199f3c0-4444-7000-8000-000000000003",
    server: plexServer,
    member: sam,
    state: "shared",
    needsYou: true,
    actions: ["check", "remove"],
    plex: samOnPlex,
    libraries,
    changedAt,
  },
  making: {
    id: "0199f3c0-4444-7000-8000-000000000004",
    server: jellyfinServer,
    member: noor,
    state: "making",
    needsYou: false,
    actions: [],
    user: { username: "noor" },
    libraries,
    changedAt,
  },
  active: {
    id: "0199f3c0-4444-7000-8000-000000000005",
    server: jellyfinServer,
    member: lin,
    state: "active",
    needsYou: false,
    actions: ["remove"],
    user: { username: "lin" },
    libraries,
    changedAt,
  },
  removing: {
    id: "0199f3c0-4444-7000-8000-000000000006",
    server: plexServer,
    member: { name: "Kai Moreno", disabled: false },
    state: "removing",
    needsYou: true,
    actions: ["retryRemove"],
    plex: { username: "kaim", email: "kai.plex@example.test", plexHome: false, byHand: false },
    libraries,
    problem: { code: "plex_refused", detail: "You don't have permission to edit this share." },
    changedAt,
  },
  removed: {
    id: "0199f3c0-4444-7000-8000-000000000007",
    server: jellyfinServer,
    member: ravi,
    state: "removed",
    needsYou: false,
    actions: ["offer"],
    user: { username: "ravi" },
    libraries,
    changedAt,
  },
} satisfies Record<MediaserversAccessView["state"], MediaserversAccessView>;

const { waiting, shared, removing, offered, making, active, removed } = accessRows;

export const accessList: MediaserversAccessList = {
  rows: [waiting, shared, removing, offered, making, active, removed],
  plexPass: true,
  plexHomeUsed: false,
};

/** The Admin's access list of one Member, Lin, who has a row on every server. */
export const linAccess: MediaserversAccessList = {
  rows: [offered, active],
  offerable: [],
  plexPass: true,
  plexHomeUsed: false,
};

export const linkableUsers: MediaserversServerUser[] = [
  { id: "a1b2c3d4e5f60718293a4b5c6d7e8f90", name: "sam" },
  { id: "0f9e8d7c6b5a49382716051423324150", name: "kids" },
];

/** Sam's own rows: Plex waiting on the Admin's share, Jellyfin still to make. */
export const myRows = {
  plexWaiting: {
    id: waiting.id,
    server: { name: plexServer.name, label: "Plex", access: "plex_tv" },
    adminName: "Ada Okafor",
    state: "waiting",
    pending: true,
    actions: ["anotherPlex"],
  },
  jellyfinOffered: {
    id: "0199f3c0-4444-7000-8000-000000000008",
    server: { name: jellyfinServer.name, label: "Jellyfin", access: "users" },
    adminName: "Ada Okafor",
    state: "offered",
    pending: true,
    actions: ["makeAccount"],
  },
  jellyfinActive: {
    id: "0199f3c0-4444-7000-8000-000000000008",
    server: { name: jellyfinServer.name, label: "Jellyfin", access: "users" },
    adminName: "Ada Okafor",
    state: "active",
    pending: false,
    actions: [],
    howToWatch: { app: "Jellyfin", signInAs: "sam", address: "https://den.example.test" },
  },
  plexActive: {
    id: waiting.id,
    server: { name: plexServer.name, label: "Plex", access: "plex_tv" },
    adminName: "Ada Okafor",
    state: "active",
    pending: false,
    actions: [],
    howToWatch: { app: "Plex", signInAs: "samr" },
  },
} satisfies Record<string, MediaserversMyAccessRow>;

export const myAccess: MediaserversMyAccess = {
  rows: [myRows.plexWaiting, myRows.jellyfinOffered],
  plex: { username: "samr" },
  plexPassLine: false,
  mailSetUp: true,
};

export const memberPin: MediaserversMemberPin = {
  pinId: 4182,
  authUrl: "https://app.plex.tv/auth#?clientID=slipstream-member&code=7XQ2",
  expiresAt: "2026-11-25T10:15:00Z",
};

export const memberPinWaiting: MediaserversMemberPinState = { state: "waiting" };

const actedOn = (row: MediaserversAccessView, mail?: MediaserversActed["mail"]) =>
  HttpResponse.json<MediaserversActed>(mail ? { row, mail } : { row });

/**
 * Answers every operation on Members' access from these rows. Put them before
 * mediaServerHandlers, whose /media-servers/:id would take /media-servers/access.
 */
export function mediaServerAccessHandlers(mine: MediaserversMyAccess = myAccess) {
  return [
    http.get(`${api}/account/media-servers`, () => HttpResponse.json(mine)),
    http.post(`${api}/account/media-servers/plex/pin`, () =>
      HttpResponse.json(memberPin, { status: 201 }),
    ),
    http.get(`${api}/account/media-servers/plex/pin/:id`, () =>
      HttpResponse.json(memberPinWaiting),
    ),
    http.delete(`${api}/account/media-servers/plex`, () => new HttpResponse(null, { status: 204 })),
    http.post(`${api}/account/media-servers/:accessId/account`, () =>
      HttpResponse.json(myRows.jellyfinActive),
    ),
    http.post(`${api}/account/media-servers/:accessId/check`, () =>
      HttpResponse.json(myRows.plexActive),
    ),
    http.get(`${api}/media-servers/access`, ({ request }) =>
      HttpResponse.json(new URL(request.url).searchParams.has("account") ? linAccess : accessList),
    ),
    http.get(`${api}/media-servers/:id/users`, () => HttpResponse.json({ users: linkableUsers })),
    http.put(`${api}/accounts/:id/media-servers/:serverId`, () => actedOn(offered)),
    http.post(`${api}/media-servers/access/:id/share`, () =>
      actedOn(
        { ...waiting, state: "active", needsYou: false, actions: ["remove"] },
        { outcome: "sent" },
      ),
    ),
    http.post(`${api}/media-servers/access/:id/mark-added`, () =>
      actedOn(
        {
          ...waiting,
          state: "active",
          needsYou: false,
          actions: ["remove"],
          plex: { ...samOnPlex, byHand: true },
        },
        { outcome: "sent" },
      ),
    ),
    http.post(`${api}/media-servers/access/:id/link`, () => actedOn(active)),
    http.post(`${api}/media-servers/access/:id/check`, () =>
      actedOn({ ...shared, state: "active", needsYou: false, actions: ["remove"] }),
    ),
    http.delete(`${api}/media-servers/access/:id`, () => HttpResponse.json<MediaserversActed>({})),
    http.post(`${api}/media-servers/plex/pin/:id/replace`, () => HttpResponse.json(plexAccount)),
  ];
}
