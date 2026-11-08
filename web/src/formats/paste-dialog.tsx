import { DialogActions } from "../account/dialog-actions";
import { useImpactSheet } from "../targets/impact-sheet";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { dataFont } from "../ui/data";
import { Dialog } from "../ui/dialog";
import {
  type Previewed,
  type Step,
  firstPreview,
  savingOf,
  usePasteSteps,
  withReplace,
} from "./paste-flow";
import { AppQuestion, PastePreview } from "./paste-preview";
import { pasteLabel } from "./paste-words";

function PasteBox({
  text,
  invalid,
  onText,
}: {
  text: string;
  invalid: boolean;
  onText: (text: string) => void;
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        JSON
      </span>
      <textarea
        rows={8}
        spellCheck={false}
        value={text}
        aria-invalid={invalid ? true : undefined}
        onChange={(event) => {
          onText(event.target.value);
        }}
        className={`min-h-(--input-height) w-full resize-y rounded-item bg-well px-3 py-2 text-body text-ink-1 ring-1 ring-hairline ring-inset aria-invalid:ring-fail-line in-data-[density=touch]:text-input-touch ${dataFont}`}
      />
    </label>
  );
}

function PasteActions({
  step,
  busy,
  ready,
  onCancel,
  onPreview,
  onSave,
}: {
  step: Step;
  busy: boolean;
  ready: boolean;
  onCancel: () => void;
  onPreview: () => void;
  onSave: (shown: Previewed) => void;
}) {
  return (
    <DialogActions>
      <Button disabled={busy} onClick={onCancel}>
        Cancel
      </Button>
      {step.step === "preview" ? (
        <Button
          look="primary"
          busy={busy}
          onClick={() => {
            onSave(step);
          }}
        >
          {pasteLabel(step.pasted.formats)}
        </Button>
      ) : (
        <Button
          look="primary"
          busy={busy}
          disabled={!ready || step.step === "app"}
          onClick={onPreview}
        >
          Preview
        </Button>
      )}
    </DialogActions>
  );
}

function PasteBody({ onDone }: { onDone: () => void }) {
  const { text, step, problem, busy, attempt, edit } = usePasteSteps();
  const impact = useImpactSheet();
  const onApp = (app: "radarr" | "sonarr") => {
    attempt(() => firstPreview(text, app));
  };
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">
        Paste one Custom Format&apos;s JSON or a list of them, as TRaSH Guides, Radarr and Sonarr
        export them. Scores aren&apos;t pasted. Give each format its score on a Target.
      </p>
      <PasteBox text={text} invalid={problem !== undefined} onText={edit} />
      {step.step === "app" && <AppQuestion onApp={onApp} />}
      {step.step === "preview" && (
        <PastePreview
          shown={step}
          onApp={onApp}
          onReplace={(replace) => {
            attempt(() => withReplace(text, step, replace));
          }}
        />
      )}
      {problem !== undefined && <AlertLine text={problem} />}
      <PasteActions
        step={step}
        busy={busy}
        ready={text.trim() !== ""}
        onCancel={onDone}
        onPreview={() => {
          attempt(() => firstPreview(text, undefined));
        }}
        onSave={(shown) => {
          attempt(async () => {
            await impact.save(savingOf(text, shown, onDone));
            return shown;
          });
        }}
      />
      {impact.sheet}
    </div>
  );
}

export function PasteDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog title="Paste Custom Formats" size={680} open={open} onOpenChange={onOpenChange}>
      <PasteBody
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
