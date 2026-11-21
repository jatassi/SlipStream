import type { BackupsChanged } from "./backupsChanged.ts";
import type { LiveEventBackupsChangedEvent } from "./liveEventBackupsChangedEvent.ts";

export interface LiveEventBackupsChanged {
  event: LiveEventBackupsChangedEvent;
  data: BackupsChanged;
}
