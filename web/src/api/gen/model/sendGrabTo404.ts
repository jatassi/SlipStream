import type { Problem } from "./problem.ts";
import type { SendGrabTo404Code } from "./sendGrabTo404Code.ts";

export type SendGrabTo404 = Problem & {
  code?: SendGrabTo404Code;
};
