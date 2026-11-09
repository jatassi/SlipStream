import type { NamingPreviewInputBodyDraft } from "./namingPreviewInputBodyDraft.ts";
import type { NamingTestName } from "./namingTestName.ts";

export interface NamingPreviewInputBody {
  /** Values over the stored settings by key, as a save would send them; null reads a setting's default */
  draft: NamingPreviewInputBodyDraft;
  /**
   * File names to name under the draft, each in the Work it matches
   * @maxItems 100
   * @nullable
   */
  names?: NamingTestName[] | null;
  /** A Work whose files to name under the draft. A Work of another Content Type is named under its own stored settings. */
  workId?: string;
}
