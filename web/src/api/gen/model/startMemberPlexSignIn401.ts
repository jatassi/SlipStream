import type { Problem } from "./problem.ts";
import type { StartMemberPlexSignIn401Code } from "./startMemberPlexSignIn401Code.ts";

export type StartMemberPlexSignIn401 = Problem & {
  code?: StartMemberPlexSignIn401Code;
};
