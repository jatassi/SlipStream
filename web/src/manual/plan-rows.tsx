import type { GrabsChoice, GrabsPlanRow } from "../api/gen/model";
import type { ClientPick } from "../grabs/send-to-sheet";
import { Button } from "../ui/button";
import { Checkbox } from "../ui/checkbox";
import { Details } from "../ui/details";
import { Select } from "../ui/select";
import { Lifts, ReleaseName, Result } from "./parts";
import { type Choices, type PickRow, type Results, clientOf, grabbed, routed } from "./picks";
import { ReleaseDetails } from "./release-details";

const meta = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";

/** What the row's sheet-wide choices do to it, and how it reports them back. */
export type RowActs = {
  readonly choices: Choices;
  readonly results: Results;
  readonly clients: readonly ClientPick[];
  readonly busy: boolean;
  readonly onTick: (key: string, ticked: boolean) => void;
  readonly onClient: (key: string, client: string) => void;
  readonly onPin: (key: string, pin: string) => void;
  readonly onCancelHeld: (row: PickRow, grabId: string) => void;
};

function Choice({ choice, onUse }: { choice: GrabsChoice; onUse: () => void }) {
  return (
    <li className="flex flex-col items-start gap-1.5 py-2.5">
      <ReleaseName release={choice.release} />
      {choice.note !== undefined && <p className={meta}>{choice.note}</p>}
      {choice.release.blocked !== null && <p>{choice.release.blocked}</p>}
      <Lifts lifts={choice.lifts} />
      <Details>
        <ReleaseDetails release={choice.release} />
      </Details>
      <Button size="small" onClick={onUse}>
        {choice.release.blocked === null ? "Use this" : "Remove from Blocklist and grab"}
      </Button>
    </li>
  );
}

/** Every other Release that could fill the row's first cell, ranked; using one pins it in the row. */
function OtherChoices({ row, onPin }: { row: GrabsPlanRow; onPin: RowActs["onPin"] }) {
  if (row.choices.length === 0) {
    return null;
  }
  return (
    <Details label="Other choices">
      <ul aria-label="Other choices" className="flex flex-col divide-y divide-hairline">
        {row.choices.map((choice) => (
          <Choice
            key={choice.pin}
            choice={choice}
            onUse={() => {
              onPin(row.key, choice.pin);
            }}
          />
        ))}
      </ul>
    </Details>
  );
}

function SendTo({
  row,
  routedWords,
  acts,
}: {
  row: GrabsPlanRow;
  routedWords: string;
  acts: RowActs;
}) {
  return (
    <Select
      label="Send to"
      options={[
        { value: routed, label: routedWords },
        ...acts.clients.map(({ key, label }) => ({ value: key, label })),
      ]}
      value={clientOf(acts.choices, row.key)}
      onValueChange={(client) => {
        acts.onClient(row.key, client);
      }}
    />
  );
}

/** A Grab of the plan: its tick, the Release in words, what it gets, what it gets past, and where it goes. */
function Pick({ row, acts }: { row: PickRow; acts: RowActs }) {
  const { pick } = row;
  const result = acts.results.get(row.key);
  const done = grabbed(acts.results, row.key);
  const client = clientOf(acts.choices, row.key);
  const chosen = acts.clients.find(({ key }) => key === client);
  return (
    <li className="flex items-start gap-3 py-3">
      <Checkbox
        label={pick.release.words}
        hideLabel
        checked={!done && !acts.choices.unticked.has(row.key)}
        disabled={done}
        onCheckedChange={(ticked) => {
          acts.onTick(row.key, ticked);
        }}
      />
      <div className="flex min-w-0 flex-1 flex-col items-start gap-1.5">
        <ReleaseName release={pick.release} />
        <p>{row.outcome}</p>
        {pick.release.blocked !== null && (
          <p>{`${pick.release.blocked} Grabbing it removes it from the Blocklist.`}</p>
        )}
        <Lifts lifts={pick.lifts} />
        <p className={meta}>
          {chosen === undefined ? pick.routed : `SlipStream sends it to ${chosen.name}.`}
        </p>
        {result !== undefined && (
          <Result
            result={result}
            busy={acts.busy}
            onCancelHeld={(grabId) => {
              acts.onCancelHeld(row, grabId);
            }}
          />
        )}
        <OtherChoices row={row} onPin={acts.onPin} />
        <Details>
          <div className="flex flex-col gap-3">
            <ReleaseDetails release={pick.release} />
            <SendTo row={row} routedWords={pick.routed} acts={acts} />
          </div>
        </Details>
      </div>
    </li>
  );
}

/** A cell nothing fills, with its plain cause; a soft-refused Release can still be chosen for it. */
function GapRow({ row, acts }: { row: GrabsPlanRow; acts: RowActs }) {
  return (
    <li className="flex flex-col items-start gap-1 py-3">
      <p className="text-ink-1">{row.missing}</p>
      {row.cause !== undefined && <p className="text-ink-2">{row.cause}</p>}
      {row.failure !== undefined && <p className="text-ink-2">{row.failure}</p>}
      <OtherChoices row={row} onPin={acts.onPin} />
    </li>
  );
}

const list =
  "flex flex-col divide-y divide-hairline rounded-card bg-glass px-4 ring-1 ring-hairline ring-inset";

/** The plan's Grabs in Want order, each ticked. */
export function PickRows({ rows, acts }: { rows: readonly PickRow[]; acts: RowActs }) {
  return (
    <ul aria-label="Grabs" className={list}>
      {rows.map((row) => (
        <Pick key={row.key} row={row} acts={acts} />
      ))}
    </ul>
  );
}

export function GapRows({ rows, acts }: { rows: readonly GrabsPlanRow[]; acts: RowActs }) {
  return (
    <ul aria-label="Cells nothing fills" className={list}>
      {rows.map((row) => (
        <GapRow key={row.key} row={row} acts={acts} />
      ))}
    </ul>
  );
}
