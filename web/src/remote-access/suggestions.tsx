import type { ReachabilitySuggestionView } from "../api/gen/model";
import { addExposure, dismissSuggestion } from "../api/gen/query/reachability/reachability";
import { AddressForm } from "../settings/address";
import { useAttempt } from "../targets/attempt";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { SubmitButton } from "../ui/form";
import { PublicForm } from "./public-form";
import { useRefresh } from "./use-refresh";

const idOf = (suggestion: ReachabilitySuggestionView) => suggestion.serverId ?? "ui-portal";

function Dismiss({ suggestion }: { suggestion: ReachabilitySuggestionView }) {
  const refresh = useRefresh();
  const dismiss = useAttempt(async () => {
    await dismissSuggestion(idOf(suggestion));
    await refresh();
  });
  return (
    <>
      <Button look="ghost" busy={dismiss.busy} onClick={dismiss.run}>
        Dismiss
      </Button>
      {dismiss.failure}
    </>
  );
}

/** A Media Server, with the public address it publishes filled in and editable. */
function ServerForm({ suggestion }: { suggestion: ReachabilitySuggestionView }) {
  const refresh = useRefresh();
  return (
    <PublicForm
      initial={suggestion.public}
      save={async (address) => {
        await addExposure({ kind: "media-server", serverId: idOf(suggestion), public: address });
        await refresh();
      }}
      actions={
        <div className="flex flex-wrap gap-2">
          <SubmitButton label="Add" />
          <Dismiss suggestion={suggestion} />
        </div>
      }
    />
  );
}

/** The UI and portal, at Address: asked for the https:// name outsiders use first when Address is private. */
function InterfaceForm({ suggestion }: { suggestion: ReachabilitySuggestionView }) {
  const refresh = useRefresh();
  const add = async () => {
    await addExposure({ kind: "ui-portal" });
    await refresh();
  };
  const accept = useAttempt(add);
  if (suggestion.addressPrivate) {
    return (
      <div className="flex flex-col gap-3">
        <p className="text-ink-2">
          People outside your home need an https:// Address they can reach. Enter the name they use.
        </p>
        <AddressForm onSaved={add}>
          <div className="flex flex-wrap gap-2">
            <SubmitButton label="Save and add" />
            <Dismiss suggestion={suggestion} />
          </div>
        </AddressForm>
      </div>
    );
  }
  return (
    <div className="flex flex-col items-start gap-3">
      <p className="text-ink-2">
        Players outside your home use your Address, <Data>{suggestion.public}</Data>.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button look="primary" busy={accept.busy} onClick={accept.run}>
          Add
        </Button>
        <Dismiss suggestion={suggestion} />
      </div>
      {accept.failure}
    </div>
  );
}

/** What SlipStream could check for use away from home, each to add or dismiss. */
export function Suggestions({
  suggestions,
}: {
  suggestions: readonly ReachabilitySuggestionView[];
}) {
  return (
    <ul aria-label="Suggestions" className="flex flex-col gap-3">
      {suggestions.map((suggestion) => (
        <li
          key={idOf(suggestion)}
          aria-label={suggestion.subject}
          className="flex flex-col gap-3 rounded-card bg-glass px-4 py-3 ring-1 ring-hairline ring-inset"
        >
          <h3 className="font-semibold text-ink-1">{suggestion.subject}</h3>
          {suggestion.kind === "media-server" ? (
            <ServerForm suggestion={suggestion} />
          ) : (
            <InterfaceForm suggestion={suggestion} />
          )}
        </li>
      ))}
    </ul>
  );
}
