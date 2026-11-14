import type { MediaserversRouteTest } from "./mediaserversRouteTest.ts";
import type { MediaserversTestedFallback } from "./mediaserversTestedFallback.ts";

export interface MediaserversTested {
  /** From the Controller at the fallback address */
  controller: MediaserversRouteTest;
  fallback: MediaserversTestedFallback;
  /** From the reaching Node at address */
  reaching: MediaserversRouteTest;
}
