import type { WorksDoneOutcome } from "./worksDoneOutcome.ts";
import type { WorksFailedOutcome } from "./worksFailedOutcome.ts";
import type { WorksSkippedOutcome } from "./worksSkippedOutcome.ts";

export type WorksOutcome = WorksDoneOutcome | WorksSkippedOutcome | WorksFailedOutcome;
