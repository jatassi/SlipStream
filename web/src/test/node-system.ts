import { HttpResponse, type JsonBodyType, http } from "msw";

import type {
  SettingChoiceField,
  SettingCountField,
  SettingPortField,
  SettingSwitchField,
  SystemNodeConfig,
  SystemReport,
  SystemRestarting,
} from "../api/gen/model";
import { problem } from "./handlers";
import { homeId, homeSystem } from "./nodes";

const about = {
  source: "default",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
} as const;

export const portSetting: SettingPortField = {
  ...about,
  key: "http.listen",
  label: "Port",
  kind: "port",
  applies: "restart",
  value: 8484,
  running: 8484,
  default: 8484,
};

const portsToTry: SettingCountField = {
  ...about,
  key: "http.ports_to_try",
  label: "Ports to try",
  kind: "count",
  applies: "restart",
  value: 10,
  running: 10,
  default: 10,
  min: 1,
  max: 100,
  unit: { one: "port", many: "ports" },
};

const level: SettingChoiceField = {
  ...about,
  key: "log.level",
  label: "Level",
  kind: "choice",
  value: "info",
  running: "info",
  default: "info",
  options: [
    { value: "debug", label: "Debug" },
    { value: "info", label: "Info" },
    { value: "warn", label: "Warn" },
    { value: "error", label: "Error" },
  ],
};

const count = (
  key: string,
  label: string,
  value: number,
  [min, max]: readonly [number, number],
  unit: SettingCountField["unit"],
  zero?: string,
): SettingCountField => ({
  ...about,
  key,
  label,
  kind: "count",
  value,
  running: value,
  default: value,
  min,
  max,
  unit,
  ...(zero !== undefined && { zero }),
});

const compress: SettingSwitchField = {
  ...about,
  key: "log.compress",
  label: "Compress rotated files",
  kind: "switch",
  value: true,
  running: true,
  default: true,
};

const viewerLines = count("log.viewer_lines", "Lines kept for the viewer", 2000, [100, 50_000], {
  one: "line",
  many: "lines",
});

export const homeConfig: SystemNodeConfig = {
  sections: [
    { id: "web", fields: [portSetting, portsToTry] },
    {
      id: "log",
      fields: [
        level,
        count("log.max_size_mb", "Rotate at", 10, [1, 1000], { one: "MB", many: "MB" }),
        count(
          "log.max_files",
          "Files kept",
          5,
          [0, 100],
          { one: "file", many: "files" },
          "No limit",
        ),
        count(
          "log.max_age_days",
          "Days kept",
          30,
          [0, 3650],
          { one: "day", many: "days" },
          "No limit",
        ),
        compress,
        viewerLines,
      ],
    },
  ],
};

/** A config with one Web and API field replaced. */
export const withPort = (port: Partial<SettingPortField>): SystemNodeConfig => ({
  sections: homeConfig.sections.map((section) =>
    section.id === "web"
      ? { ...section, fields: [{ ...portSetting, ...port }, ...section.fields.slice(1)] }
      : section,
  ),
});

/** A config whose running Lines kept for the viewer is `running`. */
export const withViewerLines = (running: number): SystemNodeConfig => ({
  sections: homeConfig.sections.map((section) => ({
    ...section,
    fields: section.fields.map((field) =>
      field.key === viewerLines.key ? { ...viewerLines, running } : field,
    ),
  })),
});

type Answer = SystemReport | SystemNodeConfig | "offline";

const offline = () => problem(503, "node_offline", "The Node is offline");

const answer = (found: Answer | undefined) =>
  found === undefined || found === "offline" ? offline() : HttpResponse.json(found);

export type Write = { method: string; nodeId: string; body: JsonBodyType };

/**
 * Each Node's facts and settings by id, an offline Node answering 503. A test
 * changes `state` and sends node.settings-changed, as a real write would.
 */
export function nodeSystems({
  systems = { [homeId]: homeSystem },
  configs = { [homeId]: homeConfig },
}: {
  systems?: Record<string, SystemReport | "offline">;
  configs?: Record<string, SystemNodeConfig | "offline">;
} = {}) {
  const state: {
    systems: typeof systems;
    configs: typeof configs;
    refuse?: Response;
  } = { systems, configs };
  const writes: Write[] = [];
  const handlers = [
    http.get("*/api/v1/nodes/:id/system", ({ params }) =>
      answer(state.systems[String(params["id"])]),
    ),
    http.get("*/api/v1/nodes/:id/config", ({ params }) =>
      answer(state.configs[String(params["id"])]),
    ),
    http.patch("*/api/v1/nodes/:id/config", async ({ request, params }) => {
      const nodeId = String(params["id"]);
      writes.push({ method: "PATCH", nodeId, body: await request.json() });
      return state.refuse ?? answer(state.configs[nodeId]);
    }),
  ];
  return { handlers, state, writes };
}

/** Every Node accepting a restart, answering before it stops. Records which Nodes were asked. */
export function restarts(accepted: Partial<SystemRestarting> = {}) {
  const asked: string[] = [];
  const handler = http.post("*/api/v1/nodes/:id/restart", ({ params }) => {
    asked.push(String(params["id"]));
    return HttpResponse.json(
      { startedAt: homeSystem.startedAt, boundPort: 8080, ...accepted },
      { status: 202 },
    );
  });
  return { handler, asked };
}
