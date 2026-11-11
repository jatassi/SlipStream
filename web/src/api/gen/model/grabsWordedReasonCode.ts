/**
 * Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. covers_more is a release that holds more than one Item, such as a season pack. SlipStream grabs Releases that hold more than one Item once S33 (Import pipeline) can import them. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist.
 */
export type GrabsWordedReasonCode =
  | "wrong_work"
  | "wrong_item"
  | "covers_more"
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
  | "blocklisted";
