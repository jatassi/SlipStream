import type { MediaserversPlayWorkView } from "./mediaserversPlayWorkView.ts";
import type { MediaserversSessionViewAudio } from "./mediaserversSessionViewAudio.ts";
import type { MediaserversSessionViewDecision } from "./mediaserversSessionViewDecision.ts";
import type { MediaserversSessionViewSubtitles } from "./mediaserversSessionViewSubtitles.ts";
import type { MediaserversSessionViewVideo } from "./mediaserversSessionViewVideo.ts";

export interface MediaserversSessionView {
  audio: MediaserversSessionViewAudio;
  decision: MediaserversSessionViewDecision;
  /** The server reports hardware transcoding */
  hardware: boolean;
  /** The file as the server names it */
  path: string;
  playId: string;
  player: string;
  product: string;
  /** Transcode reasons as the server gave them; Plex gives none */
  reasons: string[];
  startedAt: string;
  subtitles: MediaserversSessionViewSubtitles;
  user: string;
  video: MediaserversSessionViewVideo;
  /** The library file that plays; absent when no mapping leads to one */
  work?: MediaserversPlayWorkView;
}
