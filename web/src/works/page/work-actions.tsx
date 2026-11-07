import { type ReactElement, type ReactNode, useState } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import type { ImpactChange } from "../../targets/use-impact-save";
import { act } from "../actions/act";
import type { Subject } from "../actions/actions";
import { editionsSource } from "./editions";
import { EditionsPopover } from "./editions-popover";
import type { HeroActions } from "./hero-actions";
import type { WorkReads } from "./reads";
import { WorkDialogs } from "./work-dialogs";

type Open = "editions" | "another" | "edit" | "delete" | undefined;

export type WorkActions = {
  readonly hero: HeroActions;
  /** The matrix's own way into the Editions editor. */
  readonly wantAnother: (trigger: ReactElement) => ReactNode;
  readonly openDelete: () => void;
  readonly dialogs: ReactNode;
};

const menuOf = (subject: Subject, work: WorksWorkView, open: (next: Open) => () => void) =>
  [
    { label: "Edit…", onSelect: open("edit") },
    {
      label: "Refresh metadata",
      disabled: work.goneUpstreamAt !== null,
      onSelect: () => void act([subject], { kind: "refresh" }),
    },
    "separator",
    { label: "Delete…", tone: "danger", onSelect: open("delete") },
  ] as const satisfies HeroActions["menu"];

const popover = (
  editor: Omit<Parameters<typeof EditionsPopover>[0], "trigger" | "open" | "onOpenChange">,
  trigger: ReactElement | undefined,
  open: boolean,
  onOpenChange: (open: boolean) => void,
) => <EditionsPopover {...editor} trigger={trigger} open={open} onOpenChange={onOpenChange} />;

function useMonitored(subject: Subject, work: WorksWorkView): HeroActions["monitored"] {
  const [busy, setBusy] = useState(false);
  return {
    on: work.items.every(({ monitored }) => monitored),
    busy,
    set: (on) => {
      setBusy(true);
      void act([subject], { kind: "monitor", on }).finally(() => {
        setBusy(false);
      });
    },
  };
}

/** The hero's actions and the dialogs they open, each on this one Work. */
export function useWorkActions({
  type,
  work,
  reads: { wants, profile, listing },
  save,
  onDeleting,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  reads: WorkReads;
  save: (change: ImpactChange) => Promise<void>;
  onDeleting: () => void;
}): WorkActions {
  const [open, setOpen] = useState<Open>();
  const subject = { id: work.id, label: work.label, type };
  const opener = (next: Open) => () => {
    setOpen(next);
  };
  const editions = (which: "editions" | "another") => (trigger: ReactElement | undefined) =>
    popover({ work, wants, profile, save }, trigger, open === which, (next) => {
      setOpen(next ? which : undefined);
    });
  return {
    hero: {
      editions: editions("editions"),
      editionsFrom: editionsSource(wants),
      openEditions: opener("editions"),
      monitored: useMonitored(subject, work),
      menu: menuOf(subject, work, opener),
    },
    wantAnother: editions("another"),
    openDelete: opener("delete"),
    dialogs: (
      <WorkDialogs
        open={open === "edit" || open === "delete" ? open : undefined}
        subject={subject}
        work={work}
        files={listing.files.length}
        save={save}
        onClose={opener(undefined)}
        onDeleting={onDeleting}
      />
    ),
  };
}
