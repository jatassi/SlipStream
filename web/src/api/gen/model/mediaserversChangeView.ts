import type { MediaserversChangeViewKind } from "./mediaserversChangeViewKind.ts";
import type { MediaserversChangeViewLostItem } from "./mediaserversChangeViewLostItem.ts";
import type { MediaserversChangeViewResult } from "./mediaserversChangeViewResult.ts";
import type { MediaserversChangeViewState } from "./mediaserversChangeViewState.ts";
import type { MediaserversChangeViewStep } from "./mediaserversChangeViewStep.ts";

export interface MediaserversChangeView {
  /** @minimum 0 */
  attempts: number;
  createdAt: string;
  /** Where a moved or renamed file was */
  fromPath?: string;
  id: string;
  kind: MediaserversChangeViewKind;
  /** What a move or rename couldn't carry to the new item */
  lost: MediaserversChangeViewLostItem[];
  /** The file's path in its root; empty for a whole library */
  path: string;
  /** Set once sent */
  result?: MediaserversChangeViewResult;
  sentAt?: string;
  state: MediaserversChangeViewState;
  step: MediaserversChangeViewStep;
  /** The change as a person reads it, with its result or its last failure */
  words: string;
}
