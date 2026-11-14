import type { GetPlexSignIn422Code } from "./getPlexSignIn422Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn422 = Problem & {
  code?: GetPlexSignIn422Code;
};
