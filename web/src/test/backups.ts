import { HttpResponse, http } from "msw";

import type {
  BackupsBackup,
  BackupsBackupList,
  BackupsBackupSettings,
  BackupsDestination,
  SettingCountField,
  SettingDurationField,
  TasksSchedule,
} from "../api/gen/model";
import { getGetTaskSettingsMockHandler } from "../api/gen/query/tasks/tasks.msw";
import { problem } from "./handlers";
import { seedboxId } from "./nodes";
import { schedule } from "./tasks";

// 10:00 UTC on Nov 17, so a Backup at 04:01 UTC is today in the Controller's zone.
export const backupsNow = new Date("2026-11-17T10:00:00Z");

const day = 86_400;
const gib = 1024 ** 3;

const about = {
  source: "database",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
} as const;

const days = (key: string, label: string, value: number, max: number): SettingDurationField => ({
  ...about,
  key,
  label,
  kind: "duration",
  value: value * day,
  running: value * day,
  default: value * day,
  min: day,
  max: max * day,
  step: day,
});

const count = (key: string, label: string, value: number, max: number): SettingCountField => ({
  ...about,
  key,
  label,
  kind: "count",
  value,
  running: value,
  default: value,
  min: 1,
  max,
  unit: { one: "Backup", many: "Backups" },
});

export const backupSettings: BackupsBackupSettings = {
  settings: [
    days("tasks.backup.cadence", "Back up every", 1, 7),
    days("backups.keepScheduled", "Keep scheduled Backups for", 28, 3650),
    count("backups.keepEach", "Keep the newest Backups taken before updates and restores", 5, 100),
    count("backups.warnAfter", "Warn after this many missed Backups", 2, 30),
    {
      ...days("backups.copyWaitLimit", "Let a copy wait for downloads for up to", 1, 7),
      step: 3600,
    },
    days("backups.checkCopies", "Check each destination's newest copy every", 7, 90),
  ],
};

export const relayDestination: BackupsDestination = {
  id: "d1",
  nodeId: seedboxId,
  folder: "/var/lib/slipstream/backups",
  addedAt: "2026-11-01T09:00:00Z",
  copies: { copied: 2, copying: 0, failed: 0, waiting: 0 },
  newest: {
    backupId: "b".repeat(32),
    name: "slipstream-20261117T040100Z-2.1.0-scheduled-7k3m.tar.zst.age",
    copiedAt: "2026-11-17T04:05:00Z",
    check: { at: "2026-11-17T04:06:00Z", result: "ok" },
  },
  removing: false,
};

export const nightly: BackupsBackup = {
  id: "b".repeat(32),
  name: "slipstream-20261117T040100Z-2.1.0-scheduled-7k3m.tar.zst.age",
  takenAt: "2026-11-17T04:01:00Z",
  trigger: "scheduled",
  version: "2.1.0",
  size: 96 * 1024 ** 2,
  state: "verified",
  local: true,
  whileReconciling: false,
  copies: [
    {
      destinationId: relayDestination.id,
      state: "copied",
      copiedAt: "2026-11-17T04:05:00Z",
      check: { at: "2026-11-17T04:06:00Z", result: "ok" },
    },
  ],
};

export const olderNightly: BackupsBackup = {
  ...nightly,
  id: "c".repeat(32),
  name: "slipstream-20261113T040100Z-2.1.0-scheduled-2p9q.tar.zst.age",
  takenAt: "2026-11-13T04:01:00Z",
};

export const listOf = (over: Partial<BackupsBackupList> = {}): BackupsBackupList => ({
  backups: [nightly, olderNightly],
  failure: null,
  passphrase: { savedAt: "2026-11-02T10:00:00Z" },
  sheet: { downloadedAt: "2026-11-02T10:00:00Z", unlisted: [] },
  used: 3 * gib,
  ...over,
});

const utcSchedule: TasksSchedule = { ...schedule, window: { ...schedule.window, zone: "UTC" } };

export const words = "acid bench cobra dune ember fjord gecko haven iris jolt";

function passphraseHandlers(reconfirmed: boolean) {
  let confirmed = reconfirmed;
  return [
    http.get("*/api/v1/backups/passphrase", () =>
      confirmed
        ? HttpResponse.json({ passphrase: words })
        : problem(403, "needs_reconfirm", "Confirm it's you to continue"),
    ),
    http.post("*/api/v1/session/reconfirm", () => {
      confirmed = true;
      return new HttpResponse(null, { status: 204 });
    }),
  ];
}

export function backupsServer({
  list = listOf(),
  destinations = [relayDestination],
  reconfirmed = true,
}: {
  list?: BackupsBackupList;
  destinations?: BackupsDestination[];
  reconfirmed?: boolean;
} = {}) {
  const writes: { path: string; body: unknown }[] = [];
  const record = async (path: string, request: Request) => {
    const text = await request.text();
    writes.push({ path, body: text === "" ? null : (JSON.parse(text) as unknown) });
  };
  return {
    writes,
    handlers: [
      getGetTaskSettingsMockHandler(utcSchedule),
      http.get("*/api/v1/backups", () => HttpResponse.json(list)),
      http.get("*/api/v1/backup-destinations", () => HttpResponse.json({ destinations })),
      http.get("*/api/v1/backup-settings", () => HttpResponse.json(backupSettings)),
      ...passphraseHandlers(reconfirmed),
      http.post("*/api/v1/backups", async ({ request }) => {
        await record("backUpNow", request);
        return new HttpResponse(null, { status: 202 });
      }),
      http.post("*/api/v1/backups/passphrase", async ({ request }) => {
        await record("replace", request);
        return new HttpResponse(null, { status: 202 });
      }),
      http.post("*/api/v1/backups/passphrase/saved", async ({ request }) => {
        await record("saved", request);
        return new HttpResponse(null, { status: 204 });
      }),
      http.post("*/api/v1/backup-destinations", async ({ request }) => {
        await record("addDestination", request);
        return HttpResponse.json(
          { destination: relayDestination, room: { need: 3.1 * gib, free: 18 * gib } },
          { status: 201 },
        );
      }),
    ],
  };
}
