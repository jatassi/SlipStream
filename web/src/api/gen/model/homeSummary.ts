import type { HomeJobs } from "./homeJobs.ts";
import type { HomeNeedsYou } from "./homeNeedsYou.ts";
import type { HomeRecent } from "./homeRecent.ts";

export interface HomeSummary {
  jobs: HomeJobs;
  needsYou: HomeNeedsYou;
  recent: HomeRecent;
}
