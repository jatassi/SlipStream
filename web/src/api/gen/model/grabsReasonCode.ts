/**
 * Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. covers_more is a release that covers more than one Item, such as a season pack, and SlipStream doesn't grab those yet.
 */
export type GrabsReasonCode =
  | "wrong_work"
  | "wrong_item"
  | "covers_more"
  | "cut_not_wanted"
  | "quality_not_in_target"
  | "filter"
  | "size_below"
  | "size_above"
  | "size_not_checked"
  | "not_an_upgrade";
