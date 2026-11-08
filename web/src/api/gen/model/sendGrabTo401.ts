import type { Problem } from "./problem.ts";
import type { SendGrabTo401Code } from "./sendGrabTo401Code.ts";

export type SendGrabTo401 = Problem & {
  code?: SendGrabTo401Code;
};
