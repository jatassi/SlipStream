import type { HealthReport } from "../api/gen/model";
import { useCheckSystemHealth } from "../api/gen/query/health/health";
import { Button } from "../ui/button";
import { StatusSentence } from "../ui/status-sentence";
import { Checked, useCheckAnswered } from "./checked";
import { itemAnchor, levels } from "./levels";

// The spec's copy joins without a serial comma: storage, Nodes and scheduled tasks.
const andList = new Intl.ListFormat("en-GB", { type: "conjunction" });

export function HealthHead({ report }: { report: HealthReport }) {
  const check = useCheckSystemHealth({ mutation: useCheckAnswered() });
  return (
    <div className="mb-6">
      <div className="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
        <Status report={report} />
        <Button
          busy={check.isPending}
          onClick={() => {
            check.mutate();
          }}
        >
          Check all
        </Button>
      </div>
      {check.data !== undefined && (
        <div className="mt-3 text-ink-2">
          <Checked unreached={check.data.unreached} />
        </div>
      )}
    </div>
  );
}

function Status({ report: { first, level, categories } }: { report: HealthReport }) {
  if (first === null || level === null) {
    return (
      <div>
        <StatusSentence parts={[{ kind: "plain", text: "Everything is working." }]} />
        <p className="mt-1 text-ink-3">
          Checked {andList.format(categories.map((category) => category.phrase))}
        </p>
      </div>
    );
  }
  return (
    <StatusSentence
      parts={[
        {
          kind: levels[level].part,
          text: first.text,
          href: `#${itemAnchor(first.category, first.key)}`,
        },
      ]}
    />
  );
}
