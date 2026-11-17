import { HttpResponse, http } from "msw";

import type {
  NotificationsBell,
  NotificationsConnection,
  NotificationsEvent,
  NotificationsFeedRow,
  NotificationsProvider,
  NotificationsStatus,
  SettingDurationField,
} from "../api/gen/model";
import { housekeeping, succeededRun } from "./tasks";

export const providers: NotificationsProvider[] = [
  {
    id: "discord",
    label: "Discord",
    image: "bytes",
    fields: [
      {
        key: "webhookUrl",
        label: "Webhook URL",
        kind: "url",
        secret: true,
        optional: false,
        details: false,
        help: "In Discord, open the channel's settings, then Integrations, then Webhooks, and copy a webhook's URL.",
      },
      {
        key: "username",
        label: "Username",
        kind: "text",
        secret: false,
        optional: true,
        details: true,
        help: "The name messages show. Empty uses the webhook's own name.",
      },
    ],
  },
  {
    id: "webhook",
    label: "Webhook",
    image: "publicUrl",
    fields: [
      {
        key: "url",
        label: "URL",
        kind: "url",
        secret: true,
        optional: false,
        details: false,
        help: "SlipStream sends each message to this URL as JSON.",
      },
      {
        key: "username",
        label: "Username",
        kind: "text",
        secret: false,
        optional: true,
        details: true,
        help: "Sent with the password as Basic authentication.",
      },
      {
        key: "password",
        label: "Password",
        kind: "text",
        secret: true,
        optional: true,
        details: true,
        help: "",
      },
    ],
  },
];

const event = (
  id: string,
  label: string,
  group: NotificationsEvent["group"],
  onByDefault: boolean,
): NotificationsEvent => ({ id, label, group, onByDefault, bellByDefault: true, fixed: false });

export const events: NotificationsEvent[] = [
  event("grabbed", "Grabbed", "grabs", false),
  event("grab_failed", "Grab failed", "grabs", true),
  event("imported", "Imported", "imports", true),
  event("upgraded", "Upgraded", "imports", true),
  event("works_added", "Works added", "library", false),
  event("works_added_by_scan", "Works added by a Library scan", "library", false),
  event("works_deleted", "Works deleted", "library", false),
  event("new_health_item", "New Health item", "health", true),
  event("health_item_cleared", "Health item cleared", "health", true),
  { ...event("test", "Test", "system", false), bellByDefault: false, fixed: true },
];

export const recommendedEvents = [
  "grab_failed",
  "imported",
  "upgraded",
  "new_health_item",
  "health_item_cleared",
];

const sending: NotificationsStatus = {
  shown: "ok",
  since: null,
  until: null,
  reason: "",
  waiting: 0,
  givenUp: 0,
  lastSent: null,
};

/** A Discord connection that sends, its webhook URL saved and shown by its host. */
export const family: NotificationsConnection = {
  id: "0199f3c0-3333-7000-8000-000000000001",
  name: "Family",
  provider: "discord",
  on: true,
  events: recommendedEvents,
  warnings: false,
  posters: true,
  node: null,
  values: { username: "" },
  sealed: ["webhookUrl"],
  shown: { webhookUrl: "discord.com" },
  status: sending,
};

/** A Webhook connection with a saved optional password. */
export const hook: NotificationsConnection = {
  id: "0199f3c0-3333-7000-8000-000000000002",
  name: "Home",
  provider: "webhook",
  on: true,
  events: recommendedEvents,
  warnings: false,
  posters: true,
  node: null,
  values: { username: "sam" },
  sealed: ["url", "password"],
  shown: { url: "hooks.lan" },
  status: sending,
};

export const failing = (
  connection: NotificationsConnection,
  since: string,
): NotificationsConnection => ({
  ...connection,
  status: {
    ...sending,
    shown: "failing",
    since,
    reason: "Discord didn't respond. SlipStream retries automatically.",
    waiting: 3,
  },
});

export const notificationBell: NotificationsBell = {
  events: events.filter(({ fixed }) => !fixed).map(({ id }) => id),
  warnings: true,
};

const minute = 60;
const hour = 60 * minute;
const day = 24 * hour;

const duration = (
  key: string,
  label: string,
  value: number,
  [min, max]: readonly [number, number],
): SettingDurationField => ({
  kind: "duration",
  key,
  label,
  value,
  running: value,
  default: value,
  min,
  max,
  step: 1,
  source: "default",
  applies: "live",
  pendingRestart: false,
  isDefault: true,
});

export const notificationSettings = (): SettingDurationField[] => [
  duration("notifications.healthWait", "Wait before sending a new Health item", 5 * minute, [
    0,
    hour,
  ]),
  duration("notifications.giveUpAfter", "Give up on a message after", day, [hour, 7 * day]),
  duration("notifications.warnAfter", "Warn when a connection has failed for", 15 * minute, [
    minute,
    day,
  ]),
];

/** Keep history for, as Housekeeping's Tuning. */
export const historyKeep: SettingDurationField = {
  ...duration("history.keepFor", "Keep history for", 90 * day, [30 * day, 1095 * day]),
  description:
    "90 days is enough to look back over a season. A longer time keeps more History, and makes each Backup larger. History entries older than this are deleted during Housekeeping.",
};

export const feedRow = (seq: number, unread: boolean): NotificationsFeedRow => ({
  seq,
  at: new Date(Date.UTC(2026, 10, 14, 9, 0) + seq * 60_000).toISOString(),
  event: "imported",
  title: `Imported Heat ${seq}`,
  line: "WEBDL-1080p · 6.2 GB",
  link: `https://slipstream.example.test/works/${seq}`,
  unread,
});

/** GET /tasks/housekeeping, whose Tuning holds Keep history for. */
export const housekeepingHandler = http.get("*/api/v1/tasks/housekeeping", () =>
  HttpResponse.json({ task: housekeeping, units: [], runs: [succeededRun], tuning: [historyKeep] }),
);
