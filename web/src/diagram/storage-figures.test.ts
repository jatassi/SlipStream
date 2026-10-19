import { expect, test } from "vitest";

import type { NodesStorageVolume } from "../api/gen/model";
import { drawersOf } from "./storage-figures";

const volume = (id: string, total: number, free: number): NodesStorageVolume => ({
  id,
  name: `/mnt/${id}`,
  mount: `/mnt/${id}`,
  kind: "local",
  total,
  free,
  sampledAt: "2026-10-12T21:40:00Z",
});

test("each drawer reads a volume's used share, fullest first, four at most", () => {
  const volumes = [
    volume("a", 100, 75),
    volume("b", 100, 10),
    volume("c", 100, 50),
    volume("d", 100, 100),
    volume("e", 100, 40),
  ];
  expect(drawersOf({ roots: [], volumes })).toEqual([0.9, 0.6, 0.5, 0.25]);
});

test("a Node with no Storage figures, or a volume with no size, leaves its drawers at rest", () => {
  expect(drawersOf(undefined)).toEqual([]);
  expect(drawersOf({ roots: [], volumes: [volume("a", 0, 0)] })).toEqual([0]);
});
