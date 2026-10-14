export interface TargetsProfile {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Whether the Content Type's Works use it */
  default: boolean;
  id: string;
  name: string;
  /** Highest ranked first */
  targetIds: string[];
}
