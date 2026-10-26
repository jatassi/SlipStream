/** The metadata language and region a browser's language tags point to, where SlipStream offers them. */
export type PlaceSuggestion = { readonly language?: string; readonly region?: string };

/**
 * Each browser tag in its order of preference: its language when TMDB offers
 * that language for that region, or a bare language at its home region, such
 * as fr at fr-FR. The region is the first region subtag offered, else the
 * suggested language's own.
 */
export function suggestPlace(
  tags: readonly string[],
  languages: readonly string[],
  regions: readonly string[],
): PlaceSuggestion {
  const parsed = tags.map((tag) => new Intl.Locale(tag));
  const offered = (tag: string | undefined) => tag !== undefined && languages.includes(tag);
  const tagged = parsed
    .map((locale) =>
      locale.region === undefined ? undefined : `${locale.language}-${locale.region}`,
    )
    .find((tag) => offered(tag));
  const language =
    tagged ??
    parsed
      .map((locale) => `${locale.language}-${locale.language.toUpperCase()}`)
      .find((tag) => offered(tag));
  const region = [...parsed.map((locale) => locale.region), language?.split("-")[1]].find(
    (code) => code !== undefined && regions.includes(code),
  );
  return {
    ...(language !== undefined && { language }),
    ...(region !== undefined && { region }),
  };
}
