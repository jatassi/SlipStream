import type { GrabWaiting403Code } from "./grabWaiting403Code.ts";
import type { Problem } from "./problem.ts";

export type GrabWaiting403 = Problem & {
  code?: GrabWaiting403Code;
};
