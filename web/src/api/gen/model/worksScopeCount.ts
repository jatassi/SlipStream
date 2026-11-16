import type { WorksScopeCountScope } from "./worksScopeCountScope.ts";

export interface WorksScopeCount {
  /** @minimum 0 */
  items: number;
  scope: WorksScopeCountScope;
}
