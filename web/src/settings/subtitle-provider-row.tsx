import { useState } from "react";

import type { SubtitlesProvider, WorksContentType } from "../api/gen/model";
import { toastRefusal, useSubtitleWrites } from "../subtitles/settings-api";
import { providerLine } from "../subtitles/settings-words";
import { Button } from "../ui/button";
import { useClock } from "../ui/clock";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Meter } from "../ui/meter";
import { SwitchControl } from "../ui/switch";

function RemoveKey({ provider }: { provider: SubtitlesProvider }) {
  const { removeKey } = useSubtitleWrites();
  const [asking, setAsking] = useState(false);
  return (
    <>
      <Button
        size="small"
        look="ghost"
        onClick={() => {
          setAsking(true);
        }}
      >
        Remove key
      </Button>
      <ConfirmDialog
        title={`Remove your ${provider.name} key?`}
        confirmLabel="Remove key"
        look="fail"
        open={asking}
        onOpenChange={setAsking}
        onConfirm={() => removeKey(provider.id)}
      >
        <p>SlipStream stops searching {provider.name} until you add a key again.</p>
      </ConfirmDialog>
    </>
  );
}

function OnSwitch({ provider }: { provider: SubtitlesProvider }) {
  const { switchProvider } = useSubtitleWrites();
  return (
    <SwitchControl
      label={`Search ${provider.name}`}
      checked={provider.on}
      onCheckedChange={(on) => {
        void toastRefusal(switchProvider(provider.id, on));
      }}
    />
  );
}

/** A provider's name and switch, its state in one sentence, its downloads left and its key. */
export function ProviderBody({
  provider,
  types,
  openKey,
}: {
  provider: SubtitlesProvider;
  types: readonly WorksContentType[];
  openKey: (provider: SubtitlesProvider) => void;
}) {
  const clock = useClock();
  const downloads = provider.allowance?.downloads ?? null;
  return (
    <div className="flex flex-col gap-1.5 py-1">
      <div className="flex items-center justify-between gap-3">
        <span className="font-semibold text-ink-1">{provider.name}</span>
        <OnSwitch provider={provider} />
      </div>
      <span className="text-meta text-ink-2 in-data-[density=touch]:text-body-touch">
        {providerLine(provider, types, clock)}
      </span>
      {downloads !== null && provider.state.kind !== "off" && (
        <div className="max-w-60">
          <Meter
            label={`${provider.name} downloads left`}
            value={downloads.remaining}
            max={Math.max(downloads.limit, 1)}
            tone={downloads.remaining === 0 ? "warn" : undefined}
          />
        </div>
      )}
      {provider.needsKey && (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <Button
            size="small"
            onClick={() => {
              openKey(provider);
            }}
          >
            {provider.hasKey ? "Replace key" : "Add key"}
          </Button>
          {provider.hasKey && <RemoveKey provider={provider} />}
        </div>
      )}
    </div>
  );
}
