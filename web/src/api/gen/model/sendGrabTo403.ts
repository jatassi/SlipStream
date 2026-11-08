import type { Problem } from "./problem.ts";
import type { SendGrabTo403Code } from "./sendGrabTo403Code.ts";

export type SendGrabTo403 = Problem & {
  code?: SendGrabTo403Code;
};
