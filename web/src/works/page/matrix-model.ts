import type {
  FilesCell,
  FilesListedFile,
  GrabsGrab,
  TargetsCut,
  TargetsProfile,
  TargetsTarget,
  TargetsWants,
} from "../../api/gen/model";
import type { ImpactChange } from "../../targets/use-impact-save";

/** What the matrix's cells read besides their own state, and how it saves. */
export type MatrixShown = {
  readonly work: { readonly id: string; readonly label: string };
  readonly wants: TargetsWants;
  readonly grabs: readonly GrabsGrab[];
  readonly files: readonly FilesListedFile[];
  readonly cutName: (key: string) => string;
  readonly targetName: (id: string) => string;
  readonly save: (change: ImpactChange) => Promise<void>;
};

export type MatrixTarget = {
  readonly id: string;
  readonly name: string;
  readonly sourcing: string;
};

export type CellView = {
  readonly target: MatrixTarget;
  readonly cell: FilesCell | undefined;
};

/**
 * One wanted Edition. A listed cut no release of this Work has shown yet
 * is unseen: one quiet row, outside every count.
 */
export type MatrixRow =
  | {
      readonly kind: "edition";
      readonly edition: string;
      readonly name: string;
      readonly from: TargetsWants["from"];
      readonly cells: readonly CellView[];
    }
  | {
      readonly kind: "unseen";
      readonly edition: string;
      readonly name: string;
    };

export type Matrix = {
  readonly targets: readonly MatrixTarget[];
  readonly rows: readonly MatrixRow[];
};

const sourcingWords: {
  readonly [S in TargetsProfile["targets"][number]["sourcing"]]: string;
} = {
  download: "Downloaded",
};

export const anyCut = "any";

export const cutNameOf =
  (cuts: readonly TargetsCut[]) =>
  (key: string): string =>
    key === anyCut ? "Any cut" : (cuts.find((cut) => cut.key === key)?.name ?? key);

/** The Edition by Target matrix of a one-Item Work: its wanted Editions in list order, each Target in Profile order. */
export function matrixOf({
  cells,
  wants,
  profile,
  targets,
  cuts,
}: {
  cells: readonly FilesCell[];
  wants: TargetsWants;
  profile: TargetsProfile;
  targets: readonly TargetsTarget[];
  cuts: readonly TargetsCut[];
}): Matrix {
  const columns = profile.targets.map(({ targetId, sourcing }) => ({
    id: targetId,
    name: targets.find((target) => target.id === targetId)?.name ?? targetId,
    sourcing: sourcingWords[sourcing],
  }));
  const nameOf = cutNameOf(cuts);
  const editions = wants.editions.kind === "any" ? [anyCut] : wants.editions.cuts;
  return {
    targets: columns,
    rows: editions.map((edition): MatrixRow => {
      const held = cells.filter((cell) => cell.edition === edition);
      const name = nameOf(edition);
      return held.length === 0
        ? { kind: "unseen", edition, name }
        : {
            kind: "edition",
            edition,
            name,
            from: wants.from,
            cells: columns.map((target) => ({
              target,
              cell: held.find((cell) => cell.targetId === target.id),
            })),
          };
    }),
  };
}
