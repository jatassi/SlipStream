import type { IndexersCatalogEntry, IndexersCatalogEntryPrivacy } from "../../api/gen/model";

export type Privacy = IndexersCatalogEntryPrivacy | "all";

export type Filter = {
  readonly query: string;
  readonly privacy: Privacy;
  readonly language: string;
};

export const anyLanguage = "";

export const privacyWords: { readonly [P in IndexersCatalogEntryPrivacy]: string } = {
  private: "Private",
  "semi-private": "Semi-private",
  public: "Public",
};

const matchesText = (entry: IndexersCatalogEntry, query: string) => {
  const words = query.trim().toLowerCase();
  return (
    words === "" ||
    entry.name.toLowerCase().includes(words) ||
    entry.id.toLowerCase().includes(words) ||
    entry.description.toLowerCase().includes(words)
  );
};

/** The entries the search and language keep, before the privacy filter. */
const searched = (entries: readonly IndexersCatalogEntry[], { query, language }: Filter) =>
  entries.filter(
    (entry) =>
      matchesText(entry, query) && (language === anyLanguage || entry.language === language),
  );

/** The entries the filter keeps, with each privacy's count over the search and language. */
export function filtered(entries: readonly IndexersCatalogEntry[], filter: Filter) {
  const found = searched(entries, filter);
  const count = (privacy: IndexersCatalogEntryPrivacy) =>
    found.filter((entry) => entry.privacy === privacy).length;
  return {
    shown:
      filter.privacy === "all" ? found : found.filter((entry) => entry.privacy === filter.privacy),
    counts: {
      all: found.length,
      private: count("private"),
      "semi-private": count("semi-private"),
      public: count("public"),
    } satisfies Record<Privacy, number>,
  };
}

const languageNames = new Intl.DisplayNames(undefined, { type: "language" });

/** A language tag in words, such as "American English" for en-US. */
export function languageName(tag: string): string {
  try {
    return languageNames.of(tag) ?? tag;
  } catch {
    return tag;
  }
}

/** The catalog's languages by name. */
export const languagesOf = (entries: readonly IndexersCatalogEntry[]) =>
  [...new Set(entries.map((entry) => entry.language))]
    .map((tag) => ({ value: tag, label: languageName(tag) }))
    .toSorted((a, b) => a.label.localeCompare(b.label));
