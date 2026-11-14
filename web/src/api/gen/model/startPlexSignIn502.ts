import type { Problem } from "./problem.ts";
import type { StartPlexSignIn502Code } from "./startPlexSignIn502Code.ts";

export type StartPlexSignIn502 = Problem & {
  code?: StartPlexSignIn502Code;
};
