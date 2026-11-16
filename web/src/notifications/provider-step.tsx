import { useState } from "react";

import type { NotificationsProvider as Provider } from "../api/gen/model";
import { Button } from "../ui/button";
import { RadioGroup } from "../ui/radio";

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
      <RadioGroup
        label="Provider"
        options={providers.map((provider) => ({ value: provider.id, label: provider.label }))}
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
