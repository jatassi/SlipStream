import type {
  MovesBatchRow,
  MovesMovePreview,
  MovesMoveUnit,
  MovesPlace,
  MovesPreviewUnit,
} from "../api/gen/model";
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import { film } from "./actions";
import { homeId } from "./nodes";
import { archiveRoot, filmsRoot, mediaVolumes } from "./roots";

/** The films root and the archive a Move picks between. */
export const filmRoots = [
  getListRootsMockHandler({ roots: [filmsRoot, archiveRoot] }),
  getListVolumesMockHandler({ nodes: [mediaVolumes] }),
];

export const fromFilms: MovesPlace = {
  nodeId: homeId,
  nodeName: "home",
  root: "Films library root at /srv/media/films on home",
  rootId: filmsRoot.id,
};

export const toArchive: MovesPlace = {
  nodeId: homeId,
  nodeName: "home",
  root: "Films library root at /mnt/archive/films on home",
  rootId: archiveRoot.id,
};

const unit = (
  n: number,
  verdict: MovesPreviewUnit["verdict"],
  words?: string,
): MovesPreviewUnit => ({
  key: `work:${film(n).id}`,
  kind: "work",
  name: film(n).label,
  workId: film(n).id,
  from: fromFilms,
  to: toArchive,
  verdict,
  bytes: verdict === "red" ? 0 : 4 * 1024 ** 3,
  reasons:
    words === undefined
      ? []
      : [
          {
            check: "space",
            code: `${verdict}_reason`,
            color: verdict === "red" ? "red" : "amber",
            words,
          },
        ],
});

/** Three films to the archive: one that can't move, one with a note and one that moves. */
export const threeFilms: MovesMovePreview = {
  token: "plan-1",
  expiresAt: "2099-01-01T00:00:00Z",
  recommendation: "Move 2 of 3 films to home. 1 can't move: Film 3.",
  button: "Move 2 of 3",
  movable: 2,
  total: 3,
  homed: 0,
  units: [
    unit(1, "green"),
    unit(2, "amber", "/mnt/archive on home would go below its Warning level."),
    unit(
      3,
      "red",
      "/mnt/archive on home would go below its Error level. Free space there or pick another root.",
    ),
  ],
};

export const moveUnit = (n: number, over: Partial<MovesMoveUnit> = {}): MovesMoveUnit => ({
  id: `0199f0b2-2222-7000-8000-${String(n).padStart(12, "0")}`,
  batchId: "0199f0b1-1111-7000-8000-000000000001",
  kind: "work",
  contentType: "cinema",
  name: film(n).label,
  workId: film(n).id,
  bytes: 4 * 1024 ** 3,
  from: fromFilms,
  to: toArchive,
  phase: "transfer",
  words: "Copying to home",
  notes: [],
  switchNow: false,
  undone: false,
  failure: null,
  ...over,
});

export const batch = (over: Partial<MovesBatchRow> = {}): MovesBatchRow => ({
  id: "0199f0b1-1111-7000-8000-000000000001",
  origin: "bulk",
  createdAt: "2026-11-01T09:00:00Z",
  counts: { transfer: 2, done: 1 },
  total: 3,
  words: "1 of 3 films switched",
  canCancel: true,
  canRetry: false,
  canMoveBack: false,
  ...over,
});
