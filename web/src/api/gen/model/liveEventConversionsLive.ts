import type { ConversionsLiveSample } from "./conversionsLiveSample.ts";
import type { LiveEventConversionsLiveEvent } from "./liveEventConversionsLiveEvent.ts";

export interface LiveEventConversionsLive {
  event: LiveEventConversionsLiveEvent;
  data: ConversionsLiveSample;
}
