import type { Problem } from "./problem.ts";
import type { SetMetadataKey409Code } from "./setMetadataKey409Code.ts";

export type SetMetadataKey409 = Problem & {
  code?: SetMetadataKey409Code;
};
