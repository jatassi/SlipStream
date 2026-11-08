import type { Problem } from "./problem.ts";
import type { SendGrabTo409Code } from "./sendGrabTo409Code.ts";

export type SendGrabTo409 = Problem & {
  code?: SendGrabTo409Code;
};
