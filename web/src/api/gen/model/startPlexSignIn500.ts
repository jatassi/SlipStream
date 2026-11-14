import type { Problem } from "./problem.ts";
import type { StartPlexSignIn500Code } from "./startPlexSignIn500Code.ts";

export type StartPlexSignIn500 = Problem & {
  code?: StartPlexSignIn500Code;
};
