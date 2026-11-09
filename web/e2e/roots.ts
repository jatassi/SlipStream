import { z } from "zod";

import { node } from "../harness/manifest.ts";
import type { World } from "./fixtures.ts";

const Roots = z.object({
  roots: z.array(
    z.object({
      id: z.string(),
      path: z.string(),
      works: z.number(),
      scan: z.object({ lastAt: z.string().nullable() }),
    }),
  ),
});

export async function rootsOf(world: World) {
  return Roots.parse((await world.api("GET", "/api/v1/roots")).body).roots;
}

export async function moviesRoot(world: World) {
  const [movies] = await rootsOf(world);
  if (movies === undefined) {
    throw new Error("the home Scenario has no Movies root");
  }
  const sep = movies.path.includes("\\") ? "\\" : "/";
  return { ...movies, sep, media: movies.path.slice(0, movies.path.lastIndexOf(sep)) };
}

export function homeId(world: World): string {
  const id = node(world.manifest, "home").nodeId;
  if (id === undefined) {
    throw new Error("the manifest has no id for home");
  }
  return id;
}
