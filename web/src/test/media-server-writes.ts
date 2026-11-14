import { HttpResponse, http } from "msw";

import type {
  MediaserversLibraries,
  MediaserversMappingInput,
  MediaserversMappingView,
  MediaserversServerView,
  MediaserversTested,
} from "../api/gen/model";
import {
  MediaserversDraftBody,
  MediaserversMappingsBody,
  MediaserversPatchBody,
} from "../api/gen/zod/slipStream.schemas.zod";
import { denJellyfin, denLibraries, mediaServerHandlers } from "./media-servers";
import { refused } from "./roots";

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
        kind: draft.kind,
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

/**
 * A Media Servers API that keeps what the sheets write: an added server joins
 * the list with Den Jellyfin's libraries, a mapping outside every library
 * folder is refused, and the sign-in creates the key named SlipStream.
 */
export function mediaServersWorld(start: MediaserversServerView[] = [denJellyfin]) {
  const list = [...start];
  const writes: MediaServerWrite[] = [];
  const write: Write = async (path, request) => {
    const text = await request.text();
    const body: unknown = text === "" ? null : JSON.parse(text);
    writes.push({ path, body });
    return body;
  };
  const handlers = [
    ...draftHandlers(list, write),
    ...serverHandlers(list, write),
    ...mediaServerHandlers(list),
  ];
  return { handlers, writes, list };
}
