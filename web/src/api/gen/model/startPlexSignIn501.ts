import type { Problem } from "./problem.ts";
import type { StartPlexSignIn501Code } from "./startPlexSignIn501Code.ts";

export type StartPlexSignIn501 = Problem & {
  code?: StartPlexSignIn501Code;
};
