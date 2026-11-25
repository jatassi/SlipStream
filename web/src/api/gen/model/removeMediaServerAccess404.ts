import type { Problem } from "./problem.ts";
import type { RemoveMediaServerAccess404Code } from "./removeMediaServerAccess404Code.ts";

export type RemoveMediaServerAccess404 = Problem & {
  code?: RemoveMediaServerAccess404Code;
};
