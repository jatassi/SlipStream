import type { GetPlexSignIn502Code } from "./getPlexSignIn502Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn502 = Problem & {
  code?: GetPlexSignIn502Code;
};
