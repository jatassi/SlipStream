import type { GetPlexSignIn501Code } from "./getPlexSignIn501Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn501 = Problem & {
  code?: GetPlexSignIn501Code;
};
