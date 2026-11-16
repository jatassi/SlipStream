/**
 * Why the release was refused, or a note on it. wrong_item is a release whose title names another Item, no Item, or none of the searched Items. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist. not_aired is a pack listing an Item that hasn't aired, its value the Item's code. tried is a release a failed Grab of the Work already tried.
 */
export type GrabsWordedReasonCode =
  | "wrong_work"
  | "wrong_item"
  | "wrong_title"
  | "cut_not_wanted"
  | "quality_not_in_target"
  | "filter"
  | "size_below"
  | "size_above"
  | "size_not_checked"
  | "score_below_minimum"
  | "not_an_upgrade"
  | "repack_other_group"
  | "blocklisted"
  | "not_aired"
  | "tried";
