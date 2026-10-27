/**
 * Null unless stopped or home_only
 * @nullable
 */
export type IndexersStatusCause =
  | "credentials_rejected"
  | "captcha_needed"
  | "challenge_unsolved"
  | "cookie_expired"
  | "definition_unloadable"
  | "account_refused"
  | "request_rejected"
  | null;
