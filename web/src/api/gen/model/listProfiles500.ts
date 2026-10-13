import type { ListProfiles500Code } from "./listProfiles500Code.ts";
import type { Problem } from "./problem.ts";

export type ListProfiles500 = Problem & {
  code?: ListProfiles500Code;
};
