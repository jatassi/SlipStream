import type { RootsRunning } from "./rootsRunning.ts";
import type { RootsStopped } from "./rootsStopped.ts";

export interface RootsScan {
  /**
   * When the last library scan of its folder finished; null until one has
   * @nullable
   */
  lastAt: string | null;
  /** Its unit of the open Library scan run while that unit runs; null otherwise */
  running: RootsRunning | null;
  /** The gone files that stopped that scan, which Mark them gone confirms; null when it didn't stop */
  stopped: RootsStopped | null;
  /** What that scan found, or why it stopped, as a sentence; empty until one finished */
  summary: string;
}
