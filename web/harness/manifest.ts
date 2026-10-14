import { readFile } from "node:fs/promises";
import { join } from "node:path";

import type { z } from "zod";

import { manifestSchema } from "./manifest.gen.ts";

export type Manifest = z.infer<typeof manifestSchema>;
export type ManifestNode = Manifest["nodes"][number];

export const manifestNode = manifestSchema.shape.nodes.element;

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

// front is the URL of target as origin reaches it, "world" for the harness.
export function front(m: Manifest, target: string, origin: string): string {
  const found = m.fronts.find((f) => f.target === target && f.node === origin);
  if (found === undefined) {
    const origins = m.fronts.filter((f) => f.target === target).map((f) => f.node);
    throw new Error(
      origins.length === 0
        ? `the manifest has no front for ${target}`
        : `the manifest has no front for ${target} as ${origin} reaches it; it has ${origins.join(", ")}`,
    );
  }
  if (found.url === undefined) {
    throw new Error(`${target} speaks no HTTP; its front is ${found.address}`);
  }
  return found.url;
}
