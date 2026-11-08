import type { TrackersOwed } from "./trackersOwed.ts";
import type { TrackersRuleShape } from "./trackersRuleShape.ts";
import type { TrackersWindowAtRisk } from "./trackersWindowAtRisk.ts";
import type { TrackersWindowState } from "./trackersWindowState.ts";

export interface TrackersWindow {
  /** Whether it is at risk of a hit and run: yes while no copy of it announces as a seeder, unknown while every copy's Node is offline */
  atRisk: TrackersWindowAtRisk;
  /**
   * Its latest miss or release; a regrab keeps it
   * @nullable
   */
  closedAt: string | null;
  /** @pattern ^[0-9a-f]{40}$ */
  infoHash: string;
  /**
   * When its Tracker's rule and the margin were met; null unless met
   * @nullable
   */
  metAt: string | null;
  /** Its latest torrent's name */
  name: string;
  /** Its torrent's first downloaded byte */
  openedAt: string;
  /** What its latest miss still owed; null before a miss */
  owed: TrackersOwed | null;
  /** What it still needs to meet its Tracker's rule and the margin; null unless it owes under a known rule */
  owes: TrackersOwed | null;
  /**
   * When it stopped seeding while it owes; null while it seeds
   * @nullable
   */
  quietSince: string | null;
  /**
   * When a torrent of it came back after a miss
   * @nullable
   */
  regrabbedAt: string | null;
  /** The rule it snapshot when it opened; null while none is known */
  rule: TrackersRuleShape | null;
  state: TrackersWindowState;
}
