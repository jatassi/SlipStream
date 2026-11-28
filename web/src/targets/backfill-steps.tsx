import type { TargetsImpact } from "../api/gen/model";
import { countWords } from "../format";
import { RadioGroup } from "../ui/radio";
import { Select } from "../ui/select";
import { SignalLine } from "../ui/signal-line";
import { type Plan, type Shown, lackingWords } from "./backfill-plan";
import { DraftField } from "./field";
import { impactLines } from "./impact-words";
import { MadeWorksTable } from "./made-works-table";
import { listed } from "./words";

type StepProps = {
  readonly plan: Plan;
  readonly impact: TargetsImpact;
  readonly shown: Shown;
  readonly onPlan: (plan: Plan) => void;
};

export function CostStep({ impact, shown }: StepProps) {
  const { nodes, unconvertible } = impact.made;
  return (
    <div className="flex flex-col gap-3">
      <ul aria-label="What changes" className="flex flex-col gap-1.5 text-ink-1">
        {impactLines(impact).map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
      {nodes.length > 0 && <p className="text-ink-2">{`They should run on ${listed(nodes)}.`}</p>}
      {unconvertible.length > 0 && (
        <SignalLine tone="warn">
          {`No Node can convert ${countWords(unconvertible.length, shown.noun)} of them today. They wait until one can.`}
        </SignalLine>
      )}
    </div>
  );
}

export function WorksStep({ plan, impact, shown: { noun }, onPlan }: StepProps) {
  return (
    <div className="flex flex-col gap-4">
      <RadioGroup
        label={`Which ${noun.many} to convert`}
        options={[
          { value: "all", label: "Convert all" },
          { value: "choose", label: `Choose ${noun.many}` },
          { value: "none", label: "Convert none" },
        ]}
        value={plan.convert}
        onValueChange={(convert) => {
          onPlan({ ...plan, convert });
        }}
      />
      {plan.convert === "choose" && (
        <MadeWorksTable
          label={`${noun.many} to convert`}
          rows={impact.made.rows}
          picked={plan.picked}
          onPicked={(picked) => {
            onPlan({ ...plan, picked });
          }}
        />
      )}
    </div>
  );
}

function RestChoice({ plan, shown, onPlan }: StepProps) {
  const chosen = shown.others.find((other) => other.id === plan.rest);
  const warning = chosen === undefined ? undefined : lackingWords(shown, chosen);
  return (
    <div className="flex flex-col gap-3">
      <RadioGroup
        label={`Where the other ${shown.noun.many} go`}
        options={[
          { value: "new", label: "A new Profile" },
          ...shown.others.map((other) => ({ value: other.id, label: other.name })),
        ]}
        value={plan.rest}
        onValueChange={(rest) => {
          onPlan({ ...plan, rest });
        }}
      />
      {plan.rest === "new" && (
        <div className="max-w-md">
          <DraftField
            label="New Profile name"
            value={plan.newName}
            error={plan.newName.trim() === "" ? "Name the new Profile." : undefined}
            onChange={(newName) => {
              onPlan({ ...plan, newName });
            }}
          />
        </div>
      )}
      {warning !== undefined && <SignalLine tone="warn">{warning}</SignalLine>}
    </div>
  );
}

export function RestStep(props: StepProps & { readonly rest: number }) {
  const { plan, shown, onPlan } = props;
  return (
    <div className="flex flex-col gap-4">
      {props.rest > 0 && <RestChoice {...props} />}
      <div className="max-w-md">
        <Select
          label={`Default Profile for new ${shown.noun.many}`}
          options={[shown.saved, ...shown.others].map((profile) => ({
            value: profile.id,
            label: shown.profileName(profile.id),
          }))}
          value={plan.defaultId}
          onValueChange={(defaultId) => {
            onPlan({ ...plan, defaultId });
          }}
        />
      </div>
    </div>
  );
}

export function StepList({ steps, at }: { steps: readonly string[]; at: number }) {
  return (
    <ol aria-label="Steps" className="flex flex-wrap gap-x-4 gap-y-1 text-meta">
      {steps.map((step, index) => (
        <li
          key={step}
          aria-current={index === at ? "step" : undefined}
          className={index === at ? "font-semibold text-ink-1" : "text-ink-3"}
        >
          {`${String(index + 1)}. ${step}`}
        </li>
      ))}
    </ol>
  );
}
