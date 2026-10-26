/** The metadata language and region a browser's language tags point to, where SlipStream offers them. */
export type PlaceSuggestion = { readonly language?: string; readonly region?: string };

// The region a language is most often spoken in, such as JP for ja or TW for zh-Hant.
const homeRegion = (locale: Intl.Locale) =>
  new Intl.Locale(locale.language, locale.script === undefined ? {} : { script: locale.script })
    .maximize().region;

/**
 * The first browser tag whose language TMDB offers picks the language: for
 * that tag's region, else the language's home region, else its first offered
 * region. The region is that tag's, else the first later tag's that has one.
 */
export function suggestPlace(
  tags: readonly string[],
  languages: readonly string[],
  regions: readonly string[],
): PlaceSuggestion {
  const parsed = tags.map((tag) => new Intl.Locale(tag));
  for (const [index, locale] of parsed.entries()) {
    const language = [locale.region, homeRegion(locale)]
      .map((region) => `${locale.language}-${region}`)
      .concat(languages.filter((offered) => offered.startsWith(`${locale.language}-`)))
      .find((tag) => languages.includes(tag));
    if (language !== undefined) {
      const region = parsed
        .slice(index)
        .map((later) => later.region)
        .find((code) => code !== undefined && regions.includes(code));
      return { language, ...(region !== undefined && { region }) };
    }
  }
  return {};
}
