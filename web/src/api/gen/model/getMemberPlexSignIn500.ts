import type { GetMemberPlexSignIn500Code } from "./getMemberPlexSignIn500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMemberPlexSignIn500 = Problem & {
  code?: GetMemberPlexSignIn500Code;
};
