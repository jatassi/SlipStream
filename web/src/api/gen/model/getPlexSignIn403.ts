import type { GetPlexSignIn403Code } from "./getPlexSignIn403Code.ts";
import type { Problem } from "./problem.ts";

export type GetPlexSignIn403 = Problem & {
  code?: GetPlexSignIn403Code;
};
