import type { GetMemberPlexSignIn401Code } from "./getMemberPlexSignIn401Code.ts";
import type { Problem } from "./problem.ts";

export type GetMemberPlexSignIn401 = Problem & {
  code?: GetMemberPlexSignIn401Code;
};
