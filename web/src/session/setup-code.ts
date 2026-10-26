import { checkSetupCode } from "../api/gen/query/firstrun/firstrun";
import { readPref, writePref } from "../shell/prefs";

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

/** A Setup code this Node took in checkSetupCode, the only way to make one. */
export type SetupCode = string & { readonly __brand: "SetupCode" };

export const setupCodeHeaders = (code: SetupCode) => ({ "X-Setup-Code": code });

/** Checks a typed, linked or stored code against the Node and returns it normalised. */
export async function checkedSetupCode(typed: string): Promise<SetupCode> {
  const normalized = normalizeSetupCode(typed);
  await checkSetupCode({ headers: { "X-Setup-Code": normalized } });
  // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the 204 above is what makes a SetupCode
  return normalized as SetupCode;
}

/** The code this tab kept from an earlier check, still to be checked again. */
export const keptSetupCode = (): string => readPref("setupCode");

export const keepSetupCode = (code: SetupCode): void => {
  writePref("setupCode", code);
};

/** Drops the code this tab kept, once the Node is claimed or joined. */
export const forgetSetupCode = (): void => {
  writePref("setupCode", "");
};
