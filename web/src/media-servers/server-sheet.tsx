import { Suspense, useState } from "react";

import type {
  MediaserversKindView as Kind,
  MediaserversServerView as Server,
  NodesNode,
} from "../api/gen/model";
import { useListMediaServerLibrariesSuspense } from "../api/gen/query/mediaservers/mediaservers";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Skeleton } from "../ui/skeleton";
import { FieldsStep } from "./fields-step";
import { KindStep } from "./kind-step";
import { MappingsEditor } from "./mappings-field";

/** Where the Add sheet stands: choosing the kind, filling its fields, then mapping the saved server's roots. */
type Step =
  | { readonly step: "kind" }
  | { readonly step: "fields"; readonly kind: Kind }
  | { readonly step: "mappings"; readonly server: Server };

function MappingsStep({ server, onDone }: { server: Server; onDone: () => void }) {
  const libraries = useListMediaServerLibrariesSuspense(server.id).data;
  return <MappingsEditor server={server} libraries={libraries} onSaved={onDone} onLater={onDone} />;
}

function titleOf(step: Step): string {
  if (step.step === "fields") {
    return `Add ${step.kind.label}`;
  }
  return step.step === "mappings" ? `Path mappings on ${step.server.name}` : "Add a Media Server";
}

function Steps({
  kinds,
  nodes,
  step,
  onStep,
  onDone,
}: {
  kinds: readonly Kind[];
  nodes: readonly NodesNode[];
  step: Step;
  onStep: (step: Step) => void;
  onDone: () => void;
}) {
  if (step.step === "kind") {
    return (
      <KindStep
        kinds={kinds}
        onNext={(kind) => {
          onStep({ step: "fields", kind });
        }}
      />
    );
  }
  if (step.step === "fields") {
    return (
      <FieldsStep
        kind={step.kind}
        nodes={nodes}
        onBack={() => {
          onStep({ step: "kind" });
        }}
        onSaved={(server) => {
          onStep({ step: "mappings", server });
        }}
      />
    );
  }
  return (
    <Suspense fallback={<Skeleton label={`Loading ${step.server.name}'s library folders`} />}>
      <MappingsStep server={step.server} onDone={onDone} />
    </Suspense>
  );
}

/**
 * The Add sheet: the kind first, then its fields, whose Save tests the
 * server from its Node and from the Controller, then the new server's Path
 * mappings.
 */
export function AddServerButton({
  kinds,
  nodes,
}: {
  kinds: readonly Kind[];
  nodes: readonly NodesNode[];
}) {
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<Step>({ step: "kind" });
  const openChange = (next: boolean) => {
    setOpen(next);
    if (next) {
      setStep({ step: "kind" });
    }
  };
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          openChange(true);
        }}
      >
        Add a Media Server
      </Button>
      <Dialog title={titleOf(step)} size={680} open={open} onOpenChange={openChange}>
        <Steps
          kinds={kinds}
          nodes={nodes}
          step={step}
          onStep={setStep}
          onDone={() => {
            openChange(false);
          }}
        />
      </Dialog>
    </>
  );
}
