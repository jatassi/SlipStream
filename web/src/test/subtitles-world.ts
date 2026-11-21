import { HttpResponse, type RequestHandler, http } from "msw";

import type {
  SettingField,
  SubtitlesLead,
  SubtitlesProfileLanguages,
  SubtitlesProvider,
} from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import {
  OrderSubtitleProvidersBody,
  SetSubtitleProviderKeyBody,
  UpdateSubtitleProviderBody,
  UpdateSubtitleSettingsBody,
} from "../api/gen/zod/subtitles/subtitles.zod";
import { nodesAre, seedbox } from "./nodes";
import { refused } from "./roots";
import {
  day,
  gestdown,
  movies,
  moviesLanguages,
  readyLead,
  slotsField,
  storageHome,
  subdl,
  subsource,
  subtitleSettings,
  tv,
} from "./subtitles";

export type SubtitlesWorld = {
  settings: SettingField[];
  lead: SubtitlesLead;
  profiles: SubtitlesProfileLanguages[];
  providers: SubtitlesProvider[];
  slots: number;
  writes: { path: string; body: unknown }[];
  /** Answers a key PUT: the provider as saved, or a refusal. */
  key: (id: string, key: string) => Response;
  handlers: RequestHandler[];
};

const spelled = (seconds: number) => {
  const [n, unit] = seconds % day === 0 ? [seconds / day, "day"] : [seconds / 3600, "hour"];
  return n === 1 ? `1 ${unit}` : `${String(n)} ${unit}s`;
};

const gapKey = "subtitles.longestGap";
const everyKey = "tasks.subtitle-search.cadence";

/** The server's refusal of a longest gap shorter than Subtitle search's interval, on the key that moved. */
function gapRefusal(settings: readonly SettingField[], values: Record<string, unknown>) {
  const next = (key: string) =>
    Number(values[key] ?? settings.find((each) => each.key === key)?.value);
  const moved = [gapKey, everyKey].find((key) => key in values);
  if (moved === undefined || next(gapKey) >= next(everyKey)) {
    return null;
  }
  const words =
    moved === gapKey
      ? `Use ${spelled(next(everyKey))} or longer. Subtitle search runs every ${spelled(next(everyKey))}.`
      : `Use ${spelled(next(gapKey))} or shorter. A subtitle that keeps coming up empty is searched for at least every ${spelled(next(gapKey))}.`;
  return refused(
    422,
    "gap_too_short",
    "Subtitle search runs less often than a subtitle that keeps coming up empty is searched for",
    { detail: words, errors: [{ location: `body.values.${moved}`, message: words }] },
  );
}

const adopted = (settings: readonly SettingField[], values: Record<string, unknown>) =>
  settings.map((each) => {
    if (!(each.key in values)) {
      return each;
    }
    const value: unknown = values[each.key] ?? each.default;
    return Object.assign(structuredClone(each), {
      value,
      running: value,
      isDefault: value === each.default,
    });
  });

async function written(world: SubtitlesWorld, request: Request): Promise<unknown> {
  const body: unknown = await request.json();
  world.writes.push({ path: new URL(request.url).pathname, body });
  return body;
}

function settingsHandlers(world: SubtitlesWorld): RequestHandler[] {
  const body = () => ({ settings: world.settings, lead: world.lead, profiles: world.profiles });
  return [
    http.get("*/api/v1/subtitles/settings", () => HttpResponse.json(body())),
    http.patch("*/api/v1/subtitles/settings", async ({ request }) => {
      const { values } = UpdateSubtitleSettingsBody.parse(await written(world, request));
      const refusal = gapRefusal(world.settings, values);
      if (refusal !== null) {
        return refusal;
      }
      world.settings = adopted(world.settings, values);
      return HttpResponse.json(body());
    }),
    http.get("*/api/v1/subtitles/nodes/:nodeId/settings", () =>
      HttpResponse.json({ settings: [slotsField(world.slots)] }),
    ),
    http.patch("*/api/v1/subtitles/nodes/:nodeId/settings", async ({ request }) => {
      const { values } = UpdateSubtitleSettingsBody.parse(await written(world, request));
      world.slots = Number(values["subtitles.mapSlots"] ?? 1);
      return HttpResponse.json({ settings: [slotsField(world.slots)] });
    }),
  ];
}

function providerHandlers(world: SubtitlesWorld): RequestHandler[] {
  const list = () => HttpResponse.json({ providers: world.providers });
  return [
    http.get("*/api/v1/subtitles/providers", list),
    http.put("*/api/v1/subtitles/providers/order", async ({ request }) => {
      const { ids } = OrderSubtitleProvidersBody.parse(await written(world, request));
      world.providers = ids.flatMap((id, index) =>
        world.providers
          .filter((each) => each.id === id)
          .map((each) => Object.assign(structuredClone(each), { position: index + 1 })),
      );
      return list();
    }),
    http.patch("*/api/v1/subtitles/providers/:id", async ({ request, params }) => {
      const { on } = UpdateSubtitleProviderBody.parse(await written(world, request));
      world.providers = world.providers.map((each) =>
        each.id === params["id"] ? Object.assign(structuredClone(each), { on }) : each,
      );
      return HttpResponse.json(world.providers.find((each) => each.id === params["id"]));
    }),
    http.put("*/api/v1/subtitles/providers/:id/key", async ({ request, params }) => {
      const { key } = SetSubtitleProviderKeyBody.parse(await written(world, request));
      return world.key(String(params["id"]), key);
    }),
  ];
}

/** The Subtitles API from one in-memory state, keeping every write, with Movies and TV and a Storage Node. */
export function subtitlesWorld(
  start: Partial<Omit<SubtitlesWorld, "handlers" | "writes">> = {},
): SubtitlesWorld {
  const world: SubtitlesWorld = {
    settings: subtitleSettings(),
    lead: readyLead,
    profiles: [moviesLanguages],
    providers: [subdl, subsource, gestdown],
    slots: 1,
    key: () => HttpResponse.json(subdl),
    ...start,
    writes: [],
    handlers: [],
  };
  world.handlers = [
    ...settingsHandlers(world),
    ...providerHandlers(world),
    nodesAre(storageHome, seedbox),
    getListContentTypesMockHandler({ contentTypes: [movies, tv] }),
  ];
  return world;
}
