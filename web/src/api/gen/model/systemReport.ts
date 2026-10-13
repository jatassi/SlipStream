import type { SystemBuild } from "./systemBuild.ts";
import type { SystemListeners } from "./systemListeners.ts";

export interface SystemReport {
  build: SystemBuild;
  dataDir: string;
  listeners: SystemListeners;
  pid: number;
  seed: string;
  startedAt: string;
}
