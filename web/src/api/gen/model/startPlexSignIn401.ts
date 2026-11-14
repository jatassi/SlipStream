import type { Problem } from "./problem.ts";
import type { StartPlexSignIn401Code } from "./startPlexSignIn401Code.ts";

export type StartPlexSignIn401 = Problem & {
  code?: StartPlexSignIn401Code;
};
