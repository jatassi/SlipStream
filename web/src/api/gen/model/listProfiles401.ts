import type { ListProfiles401Code } from "./listProfiles401Code.ts";
import type { Problem } from "./problem.ts";

export type ListProfiles401 = Problem & {
  code?: ListProfiles401Code;
};
