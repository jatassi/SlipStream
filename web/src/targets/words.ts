export const listed = (names: readonly string[], word: "and" | "or" = "and"): string =>
  names.length < 2
    ? (names[0] ?? "")
    : `${names.slice(0, -1).join(", ")} ${word} ${names.at(-1) ?? ""}`;
