import type { PlaybackBlocked } from "./playbackBlocked.ts";
import type { PlaybackSuggestionHighest } from "./playbackSuggestionHighest.ts";
import type { PlaybackSuggestionNotAudioCodecsItem } from "./playbackSuggestionNotAudioCodecsItem.ts";
import type { PlaybackSuggestionNotChannelsItem } from "./playbackSuggestionNotChannelsItem.ts";
import type { PlaybackSuggestionNotHdrItem } from "./playbackSuggestionNotHdrItem.ts";
import type { PlaybackSuggestionVideoCodecsItem } from "./playbackSuggestionVideoCodecsItem.ts";

export interface PlaybackSuggestion {
  /** Values left out because they blocked direct play */
  blocked: PlaybackBlocked[];
  highest: PlaybackSuggestionHighest;
  name: string;
  notAudioCodecs: PlaybackSuggestionNotAudioCodecsItem[];
  notChannels: PlaybackSuggestionNotChannelsItem[];
  notHdr: PlaybackSuggestionNotHdrItem[];
  /** Values left out because the player hasn't received them unchanged in the window */
  unproven: PlaybackBlocked[];
  videoCodecs: PlaybackSuggestionVideoCodecsItem[];
}
