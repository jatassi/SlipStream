import { readFile } from "node:fs/promises";

import { z } from "zod";

const SetupCodeRecord = z.object({ msg: z.literal("setup code"), code: z.string() });

// A Node that restarts Unclaimed, as a removed Worker does, logs a new code.
export async function readSetupCode(log: string): Promise<string> {
  const code = (await readFile(log, "utf8"))
    .split("\n")
    .map((line) => (line === "" ? null : SetupCodeRecord.safeParse(JSON.parse(line))))
    .findLast((record) => record?.success === true)?.data.code;
  return code ?? Promise.reject(new Error(`${log} has no setup code record`));
}
