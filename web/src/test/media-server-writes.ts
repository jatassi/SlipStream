import { HttpResponse, http } from "msw";

import type {
  MediaserversAccountView,
  MediaserversLibraries,
  MediaserversMappingInput,
  MediaserversMappingView,
  MediaserversServerView,
  MediaserversTested,
} from "../api/gen/model";
import {
  MediaserversDraftBody,
  MediaserversKind,
  MediaserversMappingsBody,
  MediaserversPatchBody,
  SettingsInputBody,
} from "../api/gen/zod/slipStream.schemas.zod";
import {
  denJellyfin,
  denLibraries,
  mediaServerHandlers,
  mediaServerSettings,
  plexAccount,
} from "./media-servers";
import { refused } from "./roots";
import { durationWords } from "../format";

const api = "/api/v1/media-servers";

export type MediaServerWrite = { readonly path: string; readonly body: unknown };

type Write = (path: string, request: Request) => Promise<unknown>;

const insideLibraries = (folder: string, libraries: MediaserversLibraries) =>
  libraries.libraries.some(({ folders }) =>
    folders.some((each) => folder === each || folder.startsWith(`${each}/`)),
  );

const tested: MediaserversTested = {
  reaching: { ok: true, words: "home reached Den Jellyfin 12.2.0." },
  controller: { ok: true, words: "The Controller reached Den Jellyfin 12.2.0." },
  fallback: "available",
};

const outsideOf = (mappings: readonly MediaserversMappingInput[]) =>
  mappings.flatMap((mapping, index) =>
    mapping.shown && !insideLibraries(mapping.serverFolder ?? "", denLibraries)
      ? [
          {
            location: `body.mappings[${index}].serverFolder`,
            message: `${mapping.serverFolder ?? ""} isn't inside a library folder on Den Jellyfin. Its library folders are /data/films, /data/archive. Add the folder to a library on the server first.`,
          },
        ]
      : [],
  );

const savedOf = (mappings: readonly MediaserversMappingInput[]): MediaserversMappingView[] =>
  denLibraries.mappings.map((view, index) => {
    const input = mappings[index];
    const saved: MediaserversMappingView = Object.assign(structuredClone(view), {
      shown: input?.shown ?? view.shown,
      serverFolder: input?.serverFolder,
      guess: false,
      confirmed: true,
    });
    delete saved.words;
    return saved;
  });

function draftHandlers(list: MediaserversServerView[], write: Write) {
  return [
    http.post(api, async ({ request }) => {
      const draft = MediaserversDraftBody.parse(await write("add", request));
      const made: MediaserversServerView = {
        ...denJellyfin,
        id: "0199f3c0-2222-7000-8000-0000000000aa",
        kind: MediaserversKind.parse(draft.kind),
        name: draft.name,
        address: draft.address ?? "",
        mappingsNeeded: 1,
        state: { level: "warn", words: `${draft.name} needs a Path mapping for Archive.` },
      };
      list.push(made);
      return HttpResponse.json(made, { status: 201 });
    }),
    http.post(`${api}/sign-in`, async ({ request }) => {
      await write("sign-in", request);
      return HttpResponse.json({
        apiKey: "5f1d2c3b4a5f6e7d8c9b0a1f2e3d4c5b",
        deviceId: "slipstream-0001",
      });
    }),
    http.put(`${api}/:id/mappings`, async ({ request }) => {
      const { mappings } = MediaserversMappingsBody.parse(await write("mappings", request));
      const outside = outsideOf(mappings);
      return outside.length > 0
        ? refused(
            422,
            "mapping_outside_libraries",
            "A folder isn't inside any of the server's library folders",
            { errors: outside },
          )
        : HttpResponse.json({ mappings: savedOf(mappings) });
    }),
  ];
}

function serverHandlers(list: MediaserversServerView[], write: Write) {
  return [
    http.patch<{ id: string }>(`${api}/:id`, async ({ request, params }) => {
      const edit = MediaserversPatchBody.parse(await write("edit", request));
      const at = list.findIndex((s) => s.id === params.id);
      const was = list[at];
      if (was === undefined) {
        return refused(404, "media_server_not_found", "No Media Server has this id");
      }
      const got = { ...was, ...(edit.name !== undefined && { name: edit.name }) };
      list[at] = got;
      return HttpResponse.json(got);
    }),
    http.post(`${api}/:id/test`, async ({ request }) => {
      await write("test", request);
      return HttpResponse.json(tested);
    }),
    http.post(`${api}/:id/refresh`, async ({ request }) => {
      await write("refresh", request);
      return new HttpResponse(null, { status: 204 });
    }),
    http.delete<{ id: string }>(`${api}/:id`, async ({ request, params }) => {
      await write("remove", request);
      list.splice(
        list.findIndex((s) => s.id === params.id),
        1,
      );
      return new HttpResponse(null, { status: 204 });
    }),
  ];
}

export const plexPin = {
  pinId: 4711,
  authUrl: "https://app.plex.tv/auth#?code=3f9c2a",
  expiresAt: "2026-11-14T10:30:00Z",
};

/**
 * plex.tv as the Controller relays it: no account until the PIN is polled
 * once more after it starts, when the PIN signs in as ada. Signing out
 * removes every Plex server.
 */
function plexHandlers(
  list: MediaserversServerView[],
  write: Write,
  start: MediaserversAccountView | null,
) {
  let account = start;
  let polls = 0;
  return [
    http.get(`${api}/plex/account`, () =>
      account === null
        ? refused(409, "plex_not_signed_in", "SlipStream isn't signed in to Plex")
        : HttpResponse.json(account),
    ),
    http.delete(`${api}/plex/account`, async ({ request }) => {
      await write("plex-sign-out", request);
      account = null;
      list.splice(0, list.length, ...list.filter((server) => server.kind !== "plex"));
      return new HttpResponse(null, { status: 204 });
    }),
    http.post(`${api}/plex/pin`, async ({ request }) => {
      await write("plex-pin", request);
      return HttpResponse.json(plexPin, { status: 201 });
    }),
    http.get(`${api}/plex/pin/${plexPin.pinId}`, () => {
      polls += 1;
      if (polls < 2) {
        return HttpResponse.json({ state: "waiting" });
      }
      account = plexAccount;
      return HttpResponse.json({ state: "signed_in" });
    }),
  ];
}

const seconds = (value: unknown): number => {
  if (typeof value !== "number") {
    throw new TypeError("the settings fixture takes only seconds");
  }
  return value;
};

/** The six settings as saved, refusing a renewal margin no longer than the renewal schedule. */
function settingsHandlers(write: Write) {
  const settings = mediaServerSettings();
  const valueOf = (key: string) => settings.find((field) => field.key === key)?.value;
  return [
    http.get(`${api}/settings`, () => HttpResponse.json({ settings })),
    http.patch(`${api}/settings`, async ({ request }) => {
      const { values } = SettingsInputBody.parse(await write("settings", request));
      const next = (key: string) => (key in values ? seconds(values[key]) : valueOf(key));
      const margin = next("mediaServers.renewMargin");
      const every = next("tasks.plex-sign-in.cadence");
      if (margin !== undefined && every !== undefined && margin <= every) {
        const words = `"Renew when less than this is left" must be longer than "Renew Plex sign-in every", which is ${durationWords(every)}. Lengthen the margin or renew more often.`;
        return refused(
          422,
          "renew_margin_too_short",
          "The renewal margin isn't longer than the renewal schedule",
          {
            detail: words,
            errors: [{ location: "body.values.mediaServers.renewMargin", message: words }],
          },
        );
      }
      for (const field of settings) {
        if (field.key in values) {
          field.value = seconds(values[field.key]);
          field.isDefault = field.value === field.default;
        }
      }
      return HttpResponse.json({ settings });
    }),
  ];
}

/**
 * A Media Servers API that keeps what the sheets write: an added server joins
 * the list with Den Jellyfin's libraries, a mapping outside every library
 * folder is refused, the sign-in creates the key named SlipStream, a Plex PIN
 * signs in on its second poll, and the settings keep what was saved.
 */
export function mediaServersWorld(
  start: MediaserversServerView[] = [denJellyfin],
  plex: MediaserversAccountView | null = plexAccount,
) {
  const list = [...start];
  const writes: MediaServerWrite[] = [];
  const write: Write = async (path, request) => {
    const text = await request.text();
    const body: unknown = text === "" ? null : JSON.parse(text);
    writes.push({ path, body });
    return body;
  };
  const handlers = [
    ...plexHandlers(list, write, plex),
    ...settingsHandlers(write),
    ...draftHandlers(list, write),
    ...serverHandlers(list, write),
    ...mediaServerHandlers(list),
  ];
  return { handlers, writes, list };
}
