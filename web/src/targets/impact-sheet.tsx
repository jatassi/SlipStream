import { type ReactElement, useState } from "react";

import type { TargetsImpact } from "../api/gen/model";
import { dataFont } from "../ui/data";
import { Details } from "../ui/details";
import { Dialog } from "../ui/dialog";
import { RadioGroup } from "../ui/radio";
import { SignalChip } from "../ui/signal-chip";
import { AttemptActions, useAttempt } from "./attempt";
import { impactLines, moreRenamed, saveLabel } from "./impact-words";
import {
  type ImpactChange,
  type ImpactSheetState,
  type Showing,
  type Unwanted,
  useImpactSave,
} from "./use-impact-save";

const choices = [
  { value: "delete", label: "Delete" },
  { value: "keep", label: "Keep as Kept files" },
] as const;

function Changed() {
  return (
    <output className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-card bg-warn-card px-4 py-3 text-warn-text ring-1 ring-warn-line ring-inset">
      <SignalChip kind="warn">Changed</SignalChip>
      <span className="min-w-0 flex-[1_1_24ch]">
        These counts changed since the last preview. Check them before you save.
      </span>
    </output>
  );
}

function Choice({ value, onChange }: { value: Unwanted; onChange: (next: Unwanted) => void }) {
  return (
    <div className="flex flex-col gap-1.5">
      <RadioGroup
        label="Files that fill nothing"
        options={choices}
        value={value}
        onValueChange={onChange}
      />
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        A Kept file stays in the library, flagged, with &quot; (extra 1)&quot; after its name.
      </p>
    </div>
  );
}

function Renames({ impact }: { impact: TargetsImpact }) {
  const more = moreRenamed(impact);
  return (
    <Details>
      <div className="flex flex-col gap-3 pt-1">
        <ul aria-label="Renamed files" className="flex flex-col gap-3">
          {impact.renames.map((work) => (
            <li key={work.workId} className="flex flex-col gap-1">
              <span className="font-medium text-ink-1">{work.title}</span>
              <ul aria-label={work.title} className="flex flex-col gap-2">
                {work.files.map((file) => (
                  <li key={file.fileId} className="flex flex-col [overflow-wrap:anywhere]">
                    <del className={`${dataFont} text-ink-3 decoration-ink-4`}>{file.from}</del>
                    <ins className={`${dataFont} text-ink-1 no-underline`}>{file.to}</ins>
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
        {more !== undefined && <p>{more}</p>}
      </div>
    </Details>
  );
}

function ImpactBody({
  showing: { change, impact, changed },
  onSave,
  onClose,
}: {
  showing: Showing;
  onSave: (unwanted: Unwanted) => Promise<void>;
  onClose: () => void;
}) {
  const [unwanted, setUnwanted] = useState<Unwanted>("delete");
  const saving = useAttempt(() => onSave(unwanted));
  const lines = impactLines(impact);
  return (
    <div className="flex flex-col gap-4">
      {changed && <Changed />}
      <ul aria-label="What changes" className="flex flex-col gap-1.5 text-ink-1">
        {lines.length === 0 ? (
          <li>Nothing else changes.</li>
        ) : (
          lines.map((line) => <li key={line}>{line}</li>)
        )}
      </ul>
      {impact.renames.length > 0 && <Renames impact={impact} />}
      {(impact.unwanted.files > 0 || impact.downloading > 0) && (
        <Choice value={unwanted} onChange={setUnwanted} />
      )}
      <AttemptActions
        attempt={saving}
        look={unwanted === "delete" && impact.unwanted.files > 0 ? "fail" : "primary"}
        onCancel={onClose}
      >
        {saveLabel(change.action, impact, unwanted)}
      </AttemptActions>
    </div>
  );
}

function ImpactSheet({ state: { showing, onSave, onClose } }: { state: ImpactSheetState }) {
  return (
    <Dialog
      title={showing?.change.title ?? "Save"}
      open={showing !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {showing !== undefined && (
        <ImpactBody
          showing={showing}
          onSave={(unwanted) => onSave(showing, unwanted)}
          onClose={onClose}
        />
      )}
    </Dialog>
  );
}

export function useImpactSheet(): {
  readonly save: (change: ImpactChange) => Promise<void>;
  readonly sheet: ReactElement;
} {
  const { save, sheet } = useImpactSave();
  return { save, sheet: <ImpactSheet state={sheet} /> };
}
