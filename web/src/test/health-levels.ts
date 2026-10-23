import type {
  HealthNotReporting,
  HealthSettings,
  HealthVolume,
  SettingAmount,
  SettingSpaceField,
} from "../api/gen/model";
import { home } from "./nodes";

export const gib = 1024 ** 3;

const bytesOf = (amount: number): SettingAmount => ({ amount, unit: "bytes" });

const levelField = (key: string, label: string, amount: number): SettingSpaceField => ({
  key,
  label,
  kind: "space",
  source: "default",
  applies: "live",
  isDefault: true,
  pendingRestart: false,
  value: bytesOf(amount),
  running: bytesOf(amount),
  default: bytesOf(amount),
  maxBytes: 1024 ** 5,
});

/** The global levels at their defaults: a warning below 100 GB and an error below 20 GB. */
export const globalLevels: HealthSettings = {
  warnBelow: levelField("health.storageWarnBelow", "Warn when free space is below", 100 * gib),
  errorBelow: levelField(
    "health.storageErrorBelow",
    "Show an error when free space is below",
    20 * gib,
  ),
};

const following = (field: SettingSpaceField): SettingSpaceField => ({
  ...field,
  defaultFrom: "Storage levels",
});

/** /srv/media on home, following the global levels with 1.2 TB free of 8 TB. */
export const mediaLevels: HealthVolume = {
  nodeId: home.id,
  nodeName: home.name,
  mount: "/srv/media",
  name: "/srv/media",
  total: 8 * 1024 ** 4,
  free: 1.2 * 1024 ** 4,
  sampledAt: "2026-10-19T14:05:00Z",
  live: true,
  warnBelow: following(globalLevels.warnBelow),
  errorBelow: following(globalLevels.errorBelow),
  warnBytes: 100 * gib,
  errorBytes: 20 * gib,
  skipsWarning: false,
  level: null,
};

/** D:\ on home, gone since it was given a warning level of its own. */
export const goneLevels: HealthNotReporting = {
  nodeId: home.id,
  nodeName: home.name,
  mount: "D:\\",
  warnBelow: bytesOf(500 * gib),
  errorBelow: null,
};
