import type { FilesCellList } from "../api/gen/model";
import { downloading } from "./grabs";
import { orchardItemId } from "./library";
import { keeper, starter } from "./targets";
import { hdtv } from "./work-cells";

const hd = starter.id;
const uhd = keeper.id;
const base = { itemId: orchardItemId, edition: "any" } as const;

/** 1080p's upgrade downloads, its HDTV file kept until the new one imports. */
export const upgrading: FilesCellList = {
  needsYou: [],
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: false,
      state: "downloading",
      grabId: downloading.id,
      file: hdtv,
    },
    { ...base, targetId: uhd, targetName: "2160p HDR", wanted: true, state: "missing" },
  ],
  items: [{ itemId: orchardItemId, status: "missing" }],
  summary: {
    status: "missing",
    cells: { downloading: 1, missing: 1 },
    items: 1,
    itemStatus: { missing: 1 },
  },
};

/** 1080p's upgrade failed on the Grab, its HDTV file kept. */
export const failedUpgrade = (failed: string): FilesCellList => ({
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: true,
      state: "failed",
      grabId: failed,
      file: hdtv,
    },
    { ...base, targetId: uhd, targetName: "2160p HDR", wanted: true, state: "missing" },
  ],
  items: [{ itemId: orchardItemId, status: "missing" }],
  summary: {
    status: "missing",
    cells: { failed: 1, missing: 1 },
    items: 1,
    itemStatus: { missing: 1 },
  },
  needsYou: [
    {
      kind: "failed",
      itemIds: [orchardItemId],
      edition: "any",
      targetId: hd,
      waiting: [uhd],
      grabId: failed,
    },
  ],
});
