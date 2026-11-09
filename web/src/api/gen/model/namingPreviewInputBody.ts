import type { NamingPreviewInputBodyDraft } from "./namingPreviewInputBodyDraft.ts";

export interface NamingPreviewInputBody {
  /** Values over the stored settings by key, as a save would send them; null reads a setting's default */
  draft: NamingPreviewInputBodyDraft;
}
