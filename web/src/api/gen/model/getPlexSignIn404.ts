import type { GetPlexSignIn404Code } from "./getPlexSignIn404Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn404 = Problem & {
  code?: GetPlexSignIn404Code;
};
