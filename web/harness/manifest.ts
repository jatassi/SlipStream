import { readFile } from "node:fs/promises";
import { join } from "node:path";

import type { z } from "zod";

import { manifestSchema } from "./manifest.gen.ts";

export type Manifest = z.infer<typeof manifestSchema>;
export type ManifestNode = Manifest["nodes"][number];

export async function read(runDir: string): Promise<Manifest> {
  const path = join(runDir, "manifest.json");
  return manifestSchema.parse(JSON.parse(await readFile(path, "utf8")));
}

export function node(m: Manifest, name: string): ManifestNode {
  const found = m.nodes.find((n) => n.name === name);
  if (found === undefined) {
    throw new Error(
      `the manifest has no Node ${name}; it has ${m.nodes.map((n) => n.name).join(", ")}`,
    );
  }
  return found;
}

export function hostResolverRules(m: Manifest): string {
  return m.nodes.map((n) => `MAP ${n.host} ${n.address}`).join(",");
}
