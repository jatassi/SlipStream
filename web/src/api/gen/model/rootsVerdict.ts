import type { RootsRefusal } from "./rootsRefusal.ts";
import type { RootsVolume } from "./rootsVolume.ts";

export interface RootsVerdict {
  /** The folder as its Node cleaned it */
  path: string;
  /** Why it can't serve its purpose; absent when usable */
  refusal?: RootsRefusal;
  usable: boolean;
  /** The volume it sits on, when the Node read it */
  volume?: RootsVolume;
}
