import type { Problem } from "./problem.ts";
import type { StartPlexSignIn403Code } from "./startPlexSignIn403Code.ts";

export type StartPlexSignIn403 = Problem & {
  code?: StartPlexSignIn403Code;
};
