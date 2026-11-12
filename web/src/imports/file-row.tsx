import { MoreHorizontalIcon } from "@hugeicons/core-free-icons";

import type {
  ImportsImportEntry,
  ImportsImportFile,
  WorksContentType,
  WorksItem,
} from "../api/gen/model";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";
import { nameOf } from "../works/paths";
import { type Choice, choicesOf } from "./action-handlers";
import { FileDetails } from "./file-details";
// oxlint-disable-next-line import/max-dependencies -- a file row words its file, runs its actions and opens the Force confirm and the match sheet
import { useChoosing } from "./file-choosing";
import { ForceConfirm } from "./force-confirm";
import { MatchFileSheet } from "./match-file-sheet";
import { fileSentence } from "./words";

function Actions({
  name,
  primary,
  others,
  busy,
  choose,
}: {
  name: string;
  primary: Choice | undefined;
  others: readonly Choice[];
  busy: boolean;
  choose: (choice: Choice) => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {primary !== undefined && (
        <Button
          size="small"
          look="primary"
          busy={busy}
          onClick={() => {
            choose(primary);
          }}
        >
          {primary.label}
        </Button>
      )}
      {others.length > 0 && (
        <Menu
          label={`Other actions for ${name}`}
          trigger={<IconButton icon={MoreHorizontalIcon} label={`Other actions for ${name}`} />}
          items={others.map((choice) => ({
            label: choice.label,
            onSelect: () => {
              choose(choice);
            },
          }))}
        />
      )}
    </div>
  );
}

/**
 * One file that waits: the sentence it leads with, its recommended action as
 * the primary button and the others in a menu, with each Target's verdict
 * under Details.
 */
export function FileRow({
  entry,
  file,
  type,
  items,
}: {
  entry: ImportsImportEntry;
  file: ImportsImportFile;
  type: WorksContentType;
  items: readonly WorksItem[];
}) {
  const { targets } = useListTargetsSuspense().data;
  const targetName = (id: string) =>
    targets.find((target) => target.id === id)?.name ?? "A deleted Target";
  const { primary, others } = choicesOf({ file, workId: entry.workId, type, items, targetName });
  const { open, close, busy, choose, send } = useChoosing(entry, file);
  const name = nameOf(file.path);
  return (
    <li aria-label={name} className="flex min-w-0 flex-col items-start gap-2">
      <span className="font-semibold text-ink-1 wrap-anywhere">
        <Data>{name}</Data>
      </span>
      <p>{fileSentence(file, type)}</p>
      <Actions name={name} primary={primary} others={others} busy={busy} choose={choose} />
      <FileDetails file={file} targetName={targetName} />
      {open?.kind === "force" && (
        <ForceConfirm
          label={open.label}
          option={open.option}
          onForce={() => send({ action: "force", cell: open.option.cell })}
          onClose={close}
        />
      )}
      {open?.kind === "match" && (
        <MatchFileSheet entry={entry} file={file} type={type} onClose={close} />
      )}
    </li>
  );
}
