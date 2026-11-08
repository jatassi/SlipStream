import type { Problem } from "./problem.ts";
import type { SendGrabTo422Code } from "./sendGrabTo422Code.ts";

export type SendGrabTo422 = Problem & {
  code?: SendGrabTo422Code;
};
