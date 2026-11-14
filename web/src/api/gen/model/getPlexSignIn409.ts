import type { GetPlexSignIn409Code } from "./getPlexSignIn409Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn409 = Problem & {
  code?: GetPlexSignIn409Code;
};
