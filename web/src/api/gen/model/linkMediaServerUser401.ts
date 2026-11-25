import type { LinkMediaServerUser401Code } from "./linkMediaServerUser401Code.ts";
import type { Problem } from "./problem.ts";

export type LinkMediaServerUser401 = Problem & {
  code?: LinkMediaServerUser401Code;
};
