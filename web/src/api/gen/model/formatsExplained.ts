import type { FormatsEvaluated } from "./formatsEvaluated.ts";

export interface FormatsExplained {
  conditions: FormatsEvaluated[];
  /** It is the request's draft */
  draft: boolean;
  id: string;
  matched: boolean;
  name: string;
  timedOut: boolean;
}
