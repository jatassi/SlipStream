import type { Problem } from "./problem.ts";
import type { RemoveMediaServerAccess401Code } from "./removeMediaServerAccess401Code.ts";

export type RemoveMediaServerAccess401 = Problem & {
  code?: RemoveMediaServerAccess401Code;
};
