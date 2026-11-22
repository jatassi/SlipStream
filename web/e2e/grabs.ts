import { z } from "zod";

import { type World, expect } from "./fixtures.ts";

const Nodes = z.object({ nodes: z.array(z.object({ id: z.string(), name: z.string() })) });
const Added = z.object({ work: z.object({ id: z.string() }) });

export async function nodeId(world: World, name: string): Promise<string> {
  const { nodes } = Nodes.parse((await world.api("GET", "/api/v1/nodes")).body);
  const node = nodes.find((candidate) => candidate.name === name);
  if (node === undefined) {
    throw new Error(`the Scenario has no ${name}`);
  }
  return node.id;
}

// The Quiet Orchard, added and searched the way a script would, with
// seedbox's send limit holding its Transfer to about a minute.
export async function grabOnSeedbox(world: World): Promise<string> {
  const settings = `/api/v1/transfers/nodes/${await nodeId(world, "seedbox")}/settings` as const;
  const limited = await world.api("PATCH", settings, { sendLimit: 512 * 1024 });
  expect(limited.status, "seedbox's send limit is saved").toBe(200);
  const added = await world.api("POST", "/api/v1/works", {
    contentType: "movie",
    key: "9000001",
    preset: "all",
  });
  expect(added.status, "The Quiet Orchard is added").toBe(200);
  const workId = Added.parse(added.body).work.id;
  const search = await world.api("POST", `/api/v1/works/${workId}/search`);
  expect(search.status, "its search starts").toBe(202);
  return workId;
}
