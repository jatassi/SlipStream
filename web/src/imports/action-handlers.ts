import type {
  ImportsAction,
  ImportsFileAction,
  ImportsForceOption,
  ImportsImportFile,
  WorksContentType,
  WorksItem,
} from "../api/gen/model";

/** What choosing an action does: post it, confirm a Force first, open the match sheet or go to a page. */
export type Step =
  | { readonly kind: "post"; readonly body: ImportsFileAction }
  | { readonly kind: "force"; readonly option: ImportsForceOption }
  | { readonly kind: "match" }
  | { readonly kind: "editions"; readonly workId: string }
  | { readonly kind: "settings"; readonly section: string };

export type Choice = { readonly label: string; readonly step: Step };

/** What a file's action labels read from: the file, its Work and the names of its Targets. */
export type Acting = {
  readonly file: ImportsImportFile;
  readonly workId: string | null;
  readonly type: WorksContentType;
  readonly items: readonly WorksItem[];
  readonly targetName: (id: string) => string;
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

const post = (label: string, body: ImportsFileAction): Choice[] => [
  { label, step: { kind: "post", body } },
];

// "Import as episode 5", for a file whose name reads other Items than the download was for.
function confirmLabel({ file, type, items }: Acting): string {
  if (file.reason?.code !== "other_items") {
    return "Import";
  }
  const { one, many } = type.item;
  const numbers = items
    .filter(({ id }) => file.itemIds.includes(id))
    .map(({ position }) => String(position));
  return `Import as ${(numbers.length === 1 ? one : many).toLowerCase()} ${list.format(numbers)}`;
}

function workOf({ workId }: Acting): string {
  if (workId === null) {
    throw new Error("a Work's Editions are offered for an Import with no Work");
  }
  return workId;
}

/** Each action's label and what it does; Force offers one choice per Target it can force into. */
export const actionHandlers: { readonly [A in ImportsAction]: (acting: Acting) => Choice[] } = {
  confirm: (acting) => post(confirmLabel(acting), { action: "confirm" }),
  retry: () => post("Retry", { action: "retry" }),
  decline: () => post("Don't import", { action: "decline" }),
  force: ({ file, targetName }) =>
    file.forces.map((option) => ({
      label: `Force into ${targetName(option.cell.targetId)}`,
      step: { kind: "force", option },
    })),
  match: () => [{ label: "Change the match", step: { kind: "match" } }],
  notThisRelease: () => post("Not this Release", { action: "notThisRelease" }),
  changeEditions: (acting) => [
    { label: "Change the Work's Editions", step: { kind: "editions", workId: workOf(acting) } },
  ],
  turnOnRenaming: () => [
    { label: "Turn on renaming", step: { kind: "settings", section: "naming" } },
  ],
  changeSetting: () => [
    { label: "Change the setting", step: { kind: "settings", section: "importing" } },
  ],
};

/** The file's recommended choice, then the rest for its menu. */
export function choicesOf(acting: Acting): { primary: Choice | undefined; others: Choice[] } {
  const { recommended, actions } = acting.file;
  const offered = recommended === null ? [] : actionHandlers[recommended](acting);
  const [primary, ...rest] = offered;
  return {
    primary,
    others: [...rest, ...actions.flatMap((action) => actionHandlers[action](acting))],
  };
}
