import type { ReachabilityActionView, ReachabilityCard } from "../api/gen/model";
import { patchNodeNetworkSettings } from "../api/gen/query/reachability/reachability";
import { useAttempt } from "../targets/attempt";
import { Button } from "../ui/button";
import { Callout } from "../ui/callout";
import { FixText } from "./fix";
import { useRefresh } from "./use-refresh";
import { verdicts } from "./words";

export const recommendationAnchor = "recommendation";

const mapPortsKey = "reachability.mapPorts";

function useAction(action: ReachabilityActionView | null, pair: () => void) {
  const refresh = useRefresh();
  const mapPorts = useAttempt(async () => {
    if (action?.nodeId === undefined) {
      throw new Error("a map-ports action names its Node");
    }
    await patchNodeNetworkSettings(action.nodeId, { values: { [mapPortsKey]: true } });
    await refresh();
  });
  if (action === null) {
    return { button: undefined, failure: null };
  }
  const run = action.kind === "map-ports" ? mapPorts.run : pair;
  return {
    button: { label: action.label, onClick: run, busy: mapPorts.busy },
    failure: mapPorts.failure,
  };
}

/**
 * The page's one recommendation. A route that needs a relay is a red card
 * with no button until a Relay Node can be set up here, a route a failing
 * check makes Fixable an amber card with its fix's button, and a route
 * nothing outside the home tested a neutral line offering to add a Node.
 */
export function Recommendation({ card, pair }: { card: ReachabilityCard; pair: () => void }) {
  const action = useAction(card.fix.action, pair);
  const word = verdicts[card.verdict].word;
  if (card.kind === "untested") {
    return (
      <div
        id={recommendationAnchor}
        className="flex scroll-mt-20 flex-wrap items-start gap-x-4 gap-y-3 rounded-card bg-glass px-4 py-3 text-(length:--body-size) leading-(--body-line-height) ring-1 ring-hairline ring-inset"
      >
        <div className="flex min-w-0 flex-[1_1_32ch] flex-col gap-1">
          <p className="font-bold text-ink-1">{card.subject}</p>
          <div className="text-ink-2">
            <FixText fix={card.fix} />
          </div>
        </div>
        {action.button !== undefined && (
          <Button look="ghost" busy={action.button.busy} onClick={action.button.onClick}>
            {action.button.label}
          </Button>
        )}
        {action.failure}
      </div>
    );
  }
  return (
    <div id={recommendationAnchor} className="flex scroll-mt-20 flex-col gap-2">
      {card.kind === "fail" ? (
        <Callout tone="fail" chip={word} subject={card.subject}>
          <FixText fix={card.fix} />
        </Callout>
      ) : (
        <Callout
          tone="warn"
          chip={word}
          subject={card.subject}
          {...(action.button && { action: action.button })}
        >
          <FixText fix={card.fix} />
        </Callout>
      )}
      {action.failure}
    </div>
  );
}
