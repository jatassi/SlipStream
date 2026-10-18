import type { Projection } from "@anatomy/iso-kit";

import type { RoleName } from "../nodes/roles";
import type { NodeRow } from "../routes/_app/nodes/-landing/view";
import { KEYS } from "./keys.gen";
import { machineOf } from "./machines";
import type { Item } from "./parts/part";
import { CHIP_SLOTS, base, chip, lamp, plaque } from "./parts/tile";

export type TileItem = Item & { readonly role?: RoleName };

/** Every item a tile draws for a Node, in tile coordinates, unsorted. */
export function itemsOf(P: Projection, node: NodeRow): TileItem[] {
  const chips = node.roles
    .slice(0, CHIP_SLOTS)
    .map((role, slot): TileItem =>
      Object.assign(chip(P, slot, role.label, role.note), { role: role.role }),
    );
  const machines = node.roles.flatMap(({ role }) =>
    machineOf(P, role).map((part): TileItem => Object.assign(part, { role })),
  );
  return [base(P), plaque(P, node.name, node.plaque), lamp(P), ...chips, ...machines];
}

export function keyOf(name: string): number {
  const key = KEYS[name];
  if (key === undefined) {
    throw new Error(`no depth key for ${name}`);
  }
  return key;
}

/** A tile's items in painter's order. */
export const tileItems = (P: Projection, node: NodeRow): TileItem[] =>
  itemsOf(P, node).toSorted((a, b) => keyOf(a.name) - keyOf(b.name));
