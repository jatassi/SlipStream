import type { DownloadclientsOwes } from "./downloadclientsOwes.ts";
import type { DownloadclientsRatioLane } from "./downloadclientsRatioLane.ts";
import type { DownloadclientsSeedingViewAction } from "./downloadclientsSeedingViewAction.ts";
import type { DownloadclientsSeedingViewAtRisk } from "./downloadclientsSeedingViewAtRisk.ts";
import type { DownloadclientsSeedingViewClears } from "./downloadclientsSeedingViewClears.ts";
import type { DownloadclientsSeedingViewCombine } from "./downloadclientsSeedingViewCombine.ts";
import type { DownloadclientsSeedingViewEndless } from "./downloadclientsSeedingViewEndless.ts";
import type { DownloadclientsSeedingViewState } from "./downloadclientsSeedingViewState.ts";
import type { DownloadclientsTimeLane } from "./downloadclientsTimeLane.ts";

export interface DownloadclientsSeedingView {
  /** What its goal does once met, absent for a forever goal */
  action?: DownloadclientsSeedingViewAction;
  /** Whether a torrent that owes is at risk of a hit and run: yes while no copy of it announces as a seeder, unknown while every copy's Node is offline */
  atRisk: DownloadclientsSeedingViewAtRisk;
  /** What meets the rule in force, or none while no rule is known */
  clears: DownloadclientsSeedingViewClears;
  /** How its goal's ratio and seeding time combine. A forever goal seeds until an Admin removes it */
  combine: DownloadclientsSeedingViewCombine;
  /** Why its goal never acts on it, or none while its goal can act */
  endless: DownloadclientsSeedingViewEndless;
  /** Its seeding goal is met and it owes its Tracker nothing */
  goalMet: boolean;
  /** What it still needs to meet its Tracker's rule and the margin, absent unless it owes under a known rule */
  owes?: DownloadclientsOwes;
  ratio: DownloadclientsRatioLane;
  /** Its hit and run window's state, or none while it owes no Tracker */
  state: DownloadclientsSeedingViewState;
  time: DownloadclientsTimeLane;
}
