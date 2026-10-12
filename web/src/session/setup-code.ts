const readAs: Readonly<Record<string, string>> = {
  " ": "",
  "-": "",
  o: "0",
  O: "0",
  i: "1",
  I: "1",
  l: "1",
  L: "1",
};

const groupSize = 4;

export function normalizeSetupCode(typed: string): string {
  let normalized = "";
  for (const char of typed) {
    normalized += readAs[char] ?? (char >= "a" && char <= "z" ? char.toUpperCase() : char);
  }
  return normalized;
}

export function formatSetupCode(normalized: string): string {
  const chars = Array.from(normalized);
  const groups: string[] = [];
  for (let at = 0; at < chars.length; at += groupSize) {
    groups.push(chars.slice(at, at + groupSize).join(""));
  }
  return groups.join("-");
}
