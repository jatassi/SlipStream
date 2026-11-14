import type { Problem } from "./problem.ts";
import type { StartPlexSignIn409Code } from "./startPlexSignIn409Code.ts";

export type StartPlexSignIn409 = Problem & {
  code?: StartPlexSignIn409Code;
};
