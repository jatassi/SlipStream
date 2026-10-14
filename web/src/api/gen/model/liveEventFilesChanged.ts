import type { FilesChanged } from "./filesChanged.ts";
import type { LiveEventFilesChangedEvent } from "./liveEventFilesChangedEvent.ts";

export interface LiveEventFilesChanged {
  event: LiveEventFilesChangedEvent;
  data: FilesChanged;
}
