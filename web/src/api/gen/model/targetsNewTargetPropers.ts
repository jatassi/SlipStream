/**
 * prefer_and_upgrade ranks propers and repacks higher and has one replace a file of its Quality even past the cutoff. prefer_no_upgrade only ranks them higher. dont_prefer leaves them to the scores
 */
export type TargetsNewTargetPropers = "prefer_and_upgrade" | "prefer_no_upgrade" | "dont_prefer";
