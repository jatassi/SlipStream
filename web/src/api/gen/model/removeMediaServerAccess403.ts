import type { Problem } from "./problem.ts";
import type { RemoveMediaServerAccess403Code } from "./removeMediaServerAccess403Code.ts";

export type RemoveMediaServerAccess403 = Problem & {
  code?: RemoveMediaServerAccess403Code;
};
