/**
 * Why the last fetches didn't reach where the one before stopped; null when they did
 * @nullable
 */
export type IndexersRssFeedGapKind = "no_paging" | "page_cap" | "late" | null;
