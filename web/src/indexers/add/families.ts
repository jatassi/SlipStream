/** The Newznab category families by their thousand, as a site's media. */
const families: Readonly<Record<number, string>> = {
  1000: "Console",
  2000: "Movies",
  3000: "Audio",
  4000: "PC",
  5000: "TV",
  6000: "XXX",
  7000: "Books",
  8000: "Other",
};

/** The families categories fall in, in Newznab's order. */
export const familiesOf = (categories: readonly number[]): number[] =>
  [...new Set(categories.map((id) => Math.floor(id / 1000) * 1000))]
    .filter((family) => family in families)
    .toSorted((a, b) => a - b);

export const familyName = (family: number): string => families[family] ?? String(family);

/** What a site carries, such as "Movies, TV", or "" for no category. */
export const mediaWords = (categories: readonly number[]): string =>
  familiesOf(categories)
    .map((family) => familyName(family))
    .join(", ");
