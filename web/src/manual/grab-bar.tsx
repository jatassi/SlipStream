import type { ClientPick } from "../grabs/send-to-sheet";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { Details } from "../ui/details";
import { RadioGroup } from "../ui/radio";
import { Select } from "../ui/select";
import { useShell } from "../ui/shell-context";
import { routed } from "./picks";

const routedWords = "Where Tracker routing sends each one";

/** Where every row without its own client goes: a list on desktop, choices in place on phone. */
function SendAllTo({
  clients,
  value,
  onValueChange,
}: {
  clients: readonly ClientPick[];
  value: string;
  onValueChange: (client: string) => void;
}) {
  const options = [
    { value: routed, label: routedWords },
    ...clients.map(({ key, label }) => ({ value: key, label })),
  ];
  return useShell() === "phone" ? (
    <Details label="Send all to">
      <RadioGroup
        label="Send all to"
        options={options}
        value={value}
        onValueChange={onValueChange}
      />
    </Details>
  ) : (
    <Select label="Send all to" options={options} value={value} onValueChange={onValueChange} />
  );
}

/**
 * Send all to and the one Grab button for the ticked rows. A search another
 * Admin replaced takes the button's place with its sentence and a way to
 * the new one.
 */
export function GrabBar({
  count,
  busy,
  clients,
  sendAll,
  onSendAll,
  onGrab,
  failure,
  replaced,
  onReopen,
}: {
  count: number;
  busy: boolean;
  clients: readonly ClientPick[];
  sendAll: string;
  onSendAll: (client: string) => void;
  onGrab: () => void;
  failure: string | undefined;
  replaced: boolean;
  onReopen: () => void;
}) {
  if (replaced && failure !== undefined) {
    return (
      <div className="flex flex-col items-start gap-2">
        <AlertLine text={failure} />
        <Button onClick={onReopen}>Open Manual search again</Button>
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-2">
      {failure !== undefined && <AlertLine text={failure} />}
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="min-w-0 flex-[1_1_28ch]">
          <SendAllTo clients={clients} value={sendAll} onValueChange={onSendAll} />
        </div>
        <Button look="primary" disabled={count === 0} busy={busy} onClick={onGrab}>
          {`Grab ${count}`}
        </Button>
      </div>
    </div>
  );
}
