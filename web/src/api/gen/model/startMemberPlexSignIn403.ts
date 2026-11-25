import type { Problem } from "./problem.ts";
import type { StartMemberPlexSignIn403Code } from "./startMemberPlexSignIn403Code.ts";

export type StartMemberPlexSignIn403 = Problem & {
  code?: StartMemberPlexSignIn403Code;
};
