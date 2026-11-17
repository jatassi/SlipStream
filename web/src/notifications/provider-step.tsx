import { useState } from "react";

import type { NotificationsProvider as Provider } from "../api/gen/model";
import { Button } from "../ui/button";
import { RadioCards } from "../ui/radio";
import { ProviderMark } from "./provider-marks";

/** The Add sheet's first step: which provider the connection sends through. */
export function ProviderStep({
  providers,
  onNext,
}: {
  providers: readonly Provider[];
  onNext: (provider: Provider) => void;
}) {
  const [first] = providers;
  if (first === undefined) {
    throw new Error("the providers list is empty");
  }
  const [chosen, setChosen] = useState(first);
  return (
    <div className="flex flex-col gap-4">
      <RadioCards
        label="Provider"
        options={providers.map((provider) => ({
          value: provider.id,
          label: provider.label,
          mark: <ProviderMark id={provider.id} />,
        }))}
        value={chosen.id}
        onValueChange={(next) => {
          setChosen(providers.find((provider) => provider.id === next) ?? first);
        }}
      />
      <div className="flex justify-end">
        <Button
          look="primary"
          onClick={() => {
            onNext(chosen);
          }}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
