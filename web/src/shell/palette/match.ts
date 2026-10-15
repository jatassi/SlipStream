const fold = (text: string): string =>
  text.normalize("NFD").replaceAll(/\p{M}/gu, "").toLowerCase();

export const matches = (query: string, ...texts: readonly string[]): boolean => {
  const needle = fold(query.trim());
  return texts.some((text) => fold(text).includes(needle));
};
