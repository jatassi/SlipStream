import { HttpResponse, http } from "msw";

import type {
  HealthCategory,
  HealthChecked,
  HealthItem,
  HealthNotReporting,
  HealthReport,
  HealthSettings,
  HealthUnreached,
  HealthVolume,
  HealthVolumeInputBody,
  SettingAmount,
  SettingSpaceField,
} from "../api/gen/model";
import { UpdateHealthSettingsBody, UpdateHealthVolumeBody } from "../api/gen/zod/health/health.zod";
import { globalLevels } from "./health-levels";
import { seedboxId } from "./nodes";

// 06:00 UTC is a morning or an evening everywhere, so an item a few minutes
// older reads on the same day in any zone the browser has.
export const healthNow = new Date("2026-10-19T06:00:00Z");

export const minutesAgo = (minutes: number): string =>
  new Date(healthNow.getTime() - minutes * 60_000).toISOString();

const gib = 1024 ** 3;

export const fullVolume: HealthItem = {
  key: "v.home.RDo",
  subject: "D: (Media) on home",
  level: "fail",
  text: "Free up space on D: (Media) on home. It has 18 GB left.",
  since: minutesAgo(40),
  changedAt: minutesAgo(10),
  unsure: false,
  detail: [
    { label: "Free", bytes: 18 * gib },
    { label: "Error level", text: "20 GB, the global level" },
    { label: "Sampled at", at: minutesAgo(2) },
  ],
  action: { kind: "check" },
};

export const fillingVolume: HealthItem = {
  ...fullVolume,
  key: "v.home.RTo",
  subject: "E: (Archive) on home",
  level: "warn",
  text: "Free up space on E: (Archive) on home. It has 80 GB left.",
  since: minutesAgo(30),
  detail: [{ label: "Free", bytes: 80 * gib }],
};

export const awayNode: HealthItem = {
  key: seedboxId,
  subject: "seedbox",
  level: "fail",
  text: "Start seedbox or check its network. It has been offline since 05:20.",
  since: minutesAgo(40),
  changedAt: minutesAgo(40),
  unsure: false,
  detail: [],
  action: { kind: "link", href: `/nodes/${seedboxId}`, label: "Open seedbox" },
};

export const failedTask: HealthItem = {
  key: "housekeeping",
  subject: "Housekeeping",
  level: "warn",
  text: "Housekeeping failed at 04:12. Run it again.",
  since: minutesAgo(5),
  changedAt: minutesAgo(5),
  unsure: false,
  detail: [{ label: "Cause", text: "the database is locked" }],
  action: { kind: "run", taskId: "housekeeping" },
};

function worst(items: readonly HealthItem[]): HealthReport["level"] {
  if (items.length === 0) {
    return null;
  }
  return items.some((item) => item.level === "fail") ? "fail" : "warn";
}

const category = (
  id: HealthCategory["id"],
  label: string,
  phrase: string,
  canCheck: boolean,
  items: HealthItem[],
): HealthCategory => ({ id, label, phrase, canCheck, level: worst(items), items });

/** A report in registry order: Storage, Nodes, then Scheduled tasks. */
export function reportOf({
  storage = [],
  nodes = [],
  tasks = [],
}: {
  storage?: HealthItem[];
  nodes?: HealthItem[];
  tasks?: HealthItem[];
} = {}): HealthReport {
  const categories = [
    category("storage", "Storage", "storage", true, storage),
    category("nodes", "Nodes", "Nodes", true, nodes),
    category("scheduled-tasks", "Scheduled tasks", "scheduled tasks", false, tasks),
  ];
  const all = categories.flatMap((shown) =>
    shown.items.map((item) => ({ category: shown.id, item })),
  );
  const first =
    all.find(({ item }) => item.level === "fail") ?? all.find(({ item }) => item.level === "warn");
  return {
    categories,
    level: worst(all.map(({ item }) => item)),
    first:
      first === undefined
        ? null
        : { category: first.category, key: first.item.key, text: first.item.text },
  };
}

const quietReport = reportOf();

/** A Health API with nothing wrong, the levels at their defaults and no volume reporting. */
export const quietHealth = [
  http.get("*/api/v1/system/health", () => HttpResponse.json(quietReport)),
  http.get("*/api/v1/system/health/settings", () => HttpResponse.json(globalLevels)),
  http.get("*/api/v1/system/health/volumes", () =>
    HttpResponse.json({ volumes: [], notReporting: [] }),
  ),
];

function withLevel(field: SettingSpaceField, follows: SettingAmount, own: SettingAmount | null) {
  return own === null
    ? { ...field, isDefault: true, value: follows, running: follows }
    : { ...field, isDefault: false, source: "database" as const, value: own, running: own };
}

const withGlobal = (field: SettingSpaceField, own: SettingAmount | null | undefined) =>
  own === undefined ? field : withLevel(field, field.default, own);

function levelsSet(volume: HealthVolume, body: HealthVolumeInputBody, global: HealthSettings) {
  return {
    ...volume,
    ...(body.warnBelow !== undefined && {
      warnBelow: withLevel(volume.warnBelow, global.warnBelow.value, body.warnBelow),
    }),
    ...(body.errorBelow !== undefined && {
      errorBelow: withLevel(volume.errorBelow, global.errorBelow.value, body.errorBelow),
    }),
  };
}

type HealthState = {
  report: HealthReport;
  unreached: HealthUnreached[];
  settings: HealthSettings;
  volumes: HealthVolume[];
  notReporting: HealthNotReporting[];
  /** The answer to the next PATCH of the global levels, in place of saving it. */
  refusal?: Response;
};

type HealthSent = {
  checks: string[];
  settings: unknown[];
  volumes: HealthVolumeInputBody[];
  removed: string[];
};

const checksOf = (state: HealthState, sent: HealthSent) => {
  const checked = (path: string): HealthChecked => {
    sent.checks.push(path);
    return { report: state.report, unreached: state.unreached };
  };
  return [
    http.get("*/api/v1/system/health", () => HttpResponse.json(state.report)),
    http.post("*/api/v1/system/health/check", () => HttpResponse.json(checked("all"))),
    http.post("*/api/v1/system/health/:category/check", ({ params }) =>
      HttpResponse.json(checked(String(params["category"]))),
    ),
    http.post("*/api/v1/system/health/:category/items/:key/check", ({ params }) =>
      HttpResponse.json(checked(`${String(params["category"])}/${String(params["key"])}`)),
    ),
  ];
};

const levelsOf = (state: HealthState, sent: HealthSent) => [
  http.get("*/api/v1/system/health/settings", () => HttpResponse.json(state.settings)),
  http.patch("*/api/v1/system/health/settings", async ({ request }) => {
    const body = UpdateHealthSettingsBody.parse(await request.json());
    sent.settings.push(body);
    if (state.refusal !== undefined) {
      return state.refusal;
    }
    state.settings = {
      warnBelow: withGlobal(state.settings.warnBelow, body.warnBelow),
      errorBelow: withGlobal(state.settings.errorBelow, body.errorBelow),
    };
    return HttpResponse.json(state.settings);
  }),
];

const isVolume =
  (nodeId: string | null, mount: string | null) => (volume: { nodeId: string; mount: string }) =>
    volume.nodeId === nodeId && volume.mount === mount;

const volumesOf = (state: HealthState, sent: HealthSent) => [
  http.get("*/api/v1/system/health/volumes", () =>
    HttpResponse.json({ volumes: state.volumes, notReporting: state.notReporting }),
  ),
  http.patch("*/api/v1/system/health/volumes", async ({ request }) => {
    const body = UpdateHealthVolumeBody.parse(await request.json());
    sent.volumes.push(body);
    const named = isVolume(body.nodeId, body.mount);
    state.volumes = state.volumes.map((volume) =>
      named(volume) ? levelsSet(volume, body, state.settings) : volume,
    );
    return HttpResponse.json(state.volumes.find(named));
  }),
  http.delete("*/api/v1/system/health/volumes", ({ request }) => {
    const params = new URL(request.url).searchParams;
    const named = isVolume(params.get("nodeId"), params.get("mount"));
    sent.removed.push(`${params.get("nodeId")} ${params.get("mount")}`);
    state.notReporting = state.notReporting.filter((volume) => !named(volume));
    return new HttpResponse(null, { status: 204 });
  }),
];

/** The Health API over a report a test changes, recording the checks and level changes it is asked for. */
export function healthServer(initial: HealthReport) {
  const state: HealthState = {
    report: initial,
    unreached: [],
    settings: globalLevels,
    volumes: [],
    notReporting: [],
  };
  const sent: HealthSent = { checks: [], settings: [], volumes: [], removed: [] };
  const handlers = [...checksOf(state, sent), ...levelsOf(state, sent), ...volumesOf(state, sent)];
  return { handlers, state, checks: sent.checks, sent };
}
