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
  /** The Work whose file plays; absent when no mapping leads to a library file */
  workId?: string;
}
