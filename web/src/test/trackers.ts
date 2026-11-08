import { HttpResponse, http } from "msw";

import type {
  DownloadclientsTorrentRow,
  SettingDurationField,
  TrackersGoal,
  TrackersRule,
  TrackersStats,
  TrackersTrackerItem,
} from "../api/gen/model";
import {
  SetDefaultGoalBody,
  SetTrackerGoalBody,
  SetTrackerRuleBody,
  UpdateSeedingSettingsBody,
} from "../api/gen/zod/trackers/trackers.zod";
import { problem } from "./handlers";
import type { IndexerWrite } from "./indexer-plays";
import { larkspurId, torrentLeechTrackerId, trackers } from "./indexers";
import { indexersServer } from "./indexers-server";
import { torrent, torrentsServer } from "./torrents";

export const seedpoolId = "4d5e6f7a-8b9c-4d0e-9f1a-3b4c5d6e7f80";

export const statedRule: TrackersRule = {
  clears: "time",
  ratio: null,
  seedSeconds: 864_000,
  source: "releases",
};

/** A private Tracker that Larkspur reaches, whose Releases state 10 days. */
export const seedpool: TrackersTrackerItem = {
  id: seedpoolId,
  name: "seedpool",
  privacy: "private",
  hosts: ["tracker.seedpool.org"],
  keys: ["definition:seedpool"],
  indexerIds: [larkspurId],
  rule: statedRule,
  goal: null,
  counts: { torrents: 3, seeding: 2, owing: 2, atRisk: 1, goalMet: 0 },
};

export const seedpoolStats: TrackersStats = {
  uploaded: 3_221_225_472,
  downloaded: 2_147_483_648,
  ratio: 1.5,
  torrents: 3,
  seedingBytes: 1_073_741_824,
  grabs30d: 4,
  owing: 2,
};

export const defaultGoal: TrackersGoal = {
  combine: "first",
  ratio: 2,
  seedSeconds: 1_209_600,
  action: "remove-delete",
};

const duration = (
  key: string,
  label: string,
  value: number,
  min: number,
): SettingDurationField => ({
  source: "database",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
  kind: "duration",
  key,
  label,
  value,
  running: value,
  default: value,
  min,
  max: 604_800,
  step: 60,
});

const seedingSettings = [
  duration("trackers.margin", "Margin past a rule", 7200, 0),
  duration("trackers.warnAfter", "Warn when a torrent that owes hasn't announced for", 21_600, 60),
];

/** seedpool's three torrents and one of TorrentLeech's. */
export const seedpoolRows: DownloadclientsTorrentRow[] = [
  torrent({
    state: "seeding",
    name: "Hold.Fast.2024.1080p.WEB-DL",
    tabs: ["seeding", "all"],
    tracker: { id: seedpoolId, name: "seedpool" },
  }),
  torrent({
    state: "seeding",
    name: "Low.Tide.2023.1080p.WEB-DL",
    tabs: ["seeding", "all"],
    tracker: { id: seedpoolId, name: "seedpool" },
  }),
  torrent({
    state: "paused",
    name: "Salt.Line.2022.1080p.WEB-DL",
    tabs: ["all"],
    tracker: { id: seedpoolId, name: "seedpool" },
  }),
  torrent({
    state: "downloading",
    name: "Elsewhere.2021.1080p.WEB-DL",
    tabs: ["downloading", "all"],
    tracker: { id: torrentLeechTrackerId, name: "TorrentLeech" },
  }),
];

type Refusal = { field: string; detail: string };

const refused = ({ field, detail }: Refusal) =>
  HttpResponse.json(
    {
      status: 422,
      code: "seeding_value_refused",
      title: "A seeding value is out of range",
      detail,
      errors: [{ location: `body.${field}`, message: detail }],
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );

const missing = () => problem(404, "tracker_not_found", "That Tracker no longer exists.");

const inUseProblem = (detail: string) =>
  HttpResponse.json(
    { status: 409, code: "tracker_in_use", title: "This Tracker is still in use", detail },
    { status: 409, headers: { "Content-Type": "application/problem+json" } },
  );

type Seeding = { defaultGoal: TrackersGoal; settings: SettingDurationField[] };

function defaultHandlers(seeding: Seeding, writes: IndexerWrite[], refuse: Refusal | undefined) {
  return [
    http.get("*/api/v1/trackers/default-goal", () => HttpResponse.json(seeding.defaultGoal)),
    http.put("*/api/v1/trackers/default-goal", async ({ request }) => {
      const body = SetDefaultGoalBody.parse(await request.json());
      writes.push({ method: "PUT", id: "default-goal", body });
      if (refuse !== undefined) {
        return refused(refuse);
      }
      seeding.defaultGoal = body;
      return HttpResponse.json(body);
    }),
    http.get("*/api/v1/trackers/settings", () => HttpResponse.json({ settings: seeding.settings })),
    http.patch("*/api/v1/trackers/settings", async ({ request }) => {
      const body = UpdateSeedingSettingsBody.parse(await request.json());
      writes.push({ method: "PATCH", id: "settings", body });
      seeding.settings = seeding.settings.map((field) => {
        const value = body.values[field.key];
        return typeof value === "number" ? { ...field, value, isDefault: false } : field;
      });
      return HttpResponse.json({ settings: seeding.settings });
    }),
  ];
}

type State = { trackers: TrackersTrackerItem[] };

/** A handler for one Tracker that answers tracker_not_found for any other id. */
const onTracker =
  (
    state: State,
    act: (tracker: TrackersTrackerItem, request: Request) => Promise<Response> | Response,
  ) =>
  ({ params, request }: { params: Record<string, unknown>; request: Request }) => {
    const tracker = state.trackers.find((candidate) => candidate.id === params["id"]);
    return tracker === undefined ? missing() : act(tracker, request);
  };

function changer(state: State) {
  return (tracker: TrackersTrackerItem, edit: Partial<TrackersTrackerItem>) => {
    const { counts, ...saved } = { ...tracker, ...edit };
    state.trackers = state.trackers.map((other) =>
      other.id === tracker.id ? { ...saved, counts } : other,
    );
    return HttpResponse.json(saved);
  };
}

function seedingHandlers(state: State, writes: IndexerWrite[], refuse: Refusal | undefined) {
  const change = changer(state);
  return [
    http.put(
      "*/api/v1/trackers/:id/rule",
      onTracker(state, async (tracker, request) => {
        const body = SetTrackerRuleBody.parse(await request.json());
        writes.push({ method: "PUT rule", id: tracker.id, body });
        return refuse === undefined
          ? change(tracker, { rule: { ...body, source: "admin" } })
          : refused(refuse);
      }),
    ),
    http.delete(
      "*/api/v1/trackers/:id/rule",
      onTracker(state, (tracker) => {
        writes.push({ method: "DELETE rule", id: tracker.id });
        return change(tracker, { rule: statedRule });
      }),
    ),
    http.put(
      "*/api/v1/trackers/:id/goal",
      onTracker(state, async (tracker, request) => {
        const body = SetTrackerGoalBody.parse(await request.json());
        writes.push({ method: "PUT goal", id: tracker.id, body });
        return refuse === undefined ? change(tracker, { goal: body }) : refused(refuse);
      }),
    ),
    http.delete(
      "*/api/v1/trackers/:id/goal",
      onTracker(state, (tracker) => {
        writes.push({ method: "DELETE goal", id: tracker.id });
        return change(tracker, { goal: null });
      }),
    ),
  ];
}

function pageHandlers(state: State, writes: IndexerWrite[], inUse: string | undefined) {
  return [
    http.get(
      "*/api/v1/trackers/:id",
      onTracker(state, (tracker) => {
        const { counts: _, ...page } = tracker;
        return HttpResponse.json({ ...page, stats: seedpoolStats });
      }),
    ),
    http.delete(
      "*/api/v1/trackers/:id",
      onTracker(state, (tracker) => {
        writes.push({ method: "DELETE", id: tracker.id });
        if (inUse !== undefined) {
          return inUseProblem(inUse);
        }
        state.trackers = state.trackers.filter((other) => other.id !== tracker.id);
        return new HttpResponse(null, { status: 204 });
      }),
    ),
  ];
}

/**
 * The Indexers server with seedpool and the seeding side of every Tracker: a
 * rule or goal save stores it whole and is recorded in writes, unless refuse
 * names the field seeding_value_refused points at. Remove refuses with
 * tracker_in_use while inUse gives its detail.
 */
export function trackersServer({ refuse, inUse }: { refuse?: Refusal; inUse?: string } = {}) {
  const base = indexersServer({ trackerList: [...trackers, seedpool] });
  const seeding: Seeding = { defaultGoal, settings: seedingSettings };
  const handlers = [
    ...defaultHandlers(seeding, base.writes, refuse),
    ...pageHandlers(base.state, base.writes, inUse),
    ...seedingHandlers(base.state, base.writes, refuse),
    ...base.handlers,
    ...torrentsServer(seedpoolRows).handlers,
  ];
  return { handlers, state: base.state, writes: base.writes, seeding };
}
