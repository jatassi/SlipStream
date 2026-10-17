import type { SystemBuild } from "./systemBuild.ts";
import type { SystemHttpFacts } from "./systemHttpFacts.ts";
import type { SystemListeners } from "./systemListeners.ts";

export interface SystemReport {
  build: SystemBuild;
  dataDir: string;
  http: SystemHttpFacts;
  listeners: SystemListeners;
  logsDir: string;
  pid: number;
  seed: string;
  startedAt: string;
}
