import type { GetPlexSignIn401Code } from "./getPlexSignIn401Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn401 = Problem & {
  code?: GetPlexSignIn401Code;
};
