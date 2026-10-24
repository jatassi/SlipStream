import type { MetadataLastError } from "./metadataLastError.ts";
import type { MetadataReach } from "./metadataReach.ts";

export interface MetadataDetails {
  lastError: MetadataLastError | null;
  /** @nullable */
  lastSuccess: string | null;
  /** Each holder that called the provider since the Controller started or the key changed, by id */
  nodes: MetadataReach[];
}
