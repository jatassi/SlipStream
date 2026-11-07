import type { Problem } from "./problem.ts";
import type { SendGrabTo500Code } from "./sendGrabTo500Code.ts";

export type SendGrabTo500 = Problem & {
  code?: SendGrabTo500Code;
};
