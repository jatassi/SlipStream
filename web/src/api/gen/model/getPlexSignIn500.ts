import type { GetPlexSignIn500Code } from "./getPlexSignIn500Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn500 = Problem & {
  code?: GetPlexSignIn500Code;
};
