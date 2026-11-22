import type { NodesJoinCode } from "../api/gen/model";
import { useClock } from "../ui/clock";
import { Details } from "../ui/details";
import { ReadOnlyField } from "../ui/read-only-field";
import { whenWords } from "../ui/time-words";

const command = "font-mono text-meta wrap-anywhere text-ink-1 select-all";

/** The fresh code, and under Details the ways the Node takes it. */
export function Code({ code, node }: { code: NodesJoinCode; node: string }) {
  const clock = useClock();
  return (
    <div className="flex flex-col gap-4">
      <ReadOnlyField
        label="Join Code"
        value={code.code}
        description={`Expires ${whenWords(code.expiresAt, "at", clock)}. Each code works for one Node.`}
      />
      <Details>
        <ol aria-label="Ways to use the code" className="flex flex-col gap-3">
          <li className="flex flex-col gap-1">
            <span>On {node}, run:</span>
            <code className={command}>slipstream join {code.code}</code>
          </li>
          <li>
            Or open {node} in a browser. Its log gives the address and a Setup code. Enter the Setup
            code, then this Join Code.
          </li>
        </ol>
      </Details>
    </div>
  );
}
