import type { MediaserversPlayViewAudio } from "./mediaserversPlayViewAudio.ts";
import type { MediaserversPlayViewDecision } from "./mediaserversPlayViewDecision.ts";
import type { MediaserversPlayViewSubtitles } from "./mediaserversPlayViewSubtitles.ts";
import type { MediaserversPlayViewVideo } from "./mediaserversPlayViewVideo.ts";
import type { MediaserversPlayWorkView } from "./mediaserversPlayWorkView.ts";
import type { MediaserversReceivedView } from "./mediaserversReceivedView.ts";

export interface MediaserversPlayView {
  audio: MediaserversPlayViewAudio;
  decision: MediaserversPlayViewDecision;
  /**
   * null while the play is open
   * @nullable
   */
  endedAt: string | null;
  hardware: boolean;
  id: string;
  lastSeenAt: string;
  /** The file as the server names it */
  path: string;
  platform: string;
  player: string;
  playerKey: string;
  product: string;
  reasons: string[];
  received: MediaserversReceivedView;
  startedAt: string;
  subtitles: MediaserversPlayViewSubtitles;
  user: string;
  video: MediaserversPlayViewVideo;
  /** Absent while the play has no snapshot */
  work?: MediaserversPlayWorkView;
}
