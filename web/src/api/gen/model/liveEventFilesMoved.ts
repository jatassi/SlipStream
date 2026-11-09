import type { FilesMoved } from "./filesMoved.ts";
import type { LiveEventFilesMovedEvent } from "./liveEventFilesMovedEvent.ts";

export interface LiveEventFilesMoved {
  event: LiveEventFilesMovedEvent;
  data: FilesMoved;
}
