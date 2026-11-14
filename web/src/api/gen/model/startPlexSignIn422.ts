import type { Problem } from "./problem.ts";
import type { StartPlexSignIn422Code } from "./startPlexSignIn422Code.ts";

export type StartPlexSignIn422 = Problem & {
  code?: StartPlexSignIn422Code;
};
