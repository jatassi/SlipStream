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
import { PlexStep, forgetPlexPin, pendingPlexKind } from "./plex-step";

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
    const Fields = step.kind.signIn === "plex_pin" ? PlexStep : FieldsStep;
    return (
      <Suspense fallback={<Skeleton label={`Loading ${step.kind.label}`} />}>
        <Fields
          kind={step.kind}
          nodes={nodes}
          onBack={() => {
            onStep({ step: "kind" });
          }}
          onSaved={(server) => {
            onStep({ step: "mappings", server });
          }}
        />
      </Suspense>
    );
  }
  return (
    <Suspense fallback={<Skeleton label={`Loading ${step.server.name}'s library folders`} />}>
      <MappingsStep server={step.server} onDone={onDone} />
    </Suspense>
  );
}

/**
 * The Add sheet: the kind first, then its fields or the Plex sign-in, whose
 * Save tests the server from its Node and from the Controller, then the new
 * server's Path mappings. A Plex sign-in still waiting after a reload opens
 * the sheet on the Plex step again.
 */
export function AddServerButton({
  kinds,
  nodes,
}: {
  kinds: readonly Kind[];
  nodes: readonly NodesNode[];
}) {
  const [resume] = useState(() => pendingPlexKind(kinds));
  const [open, setOpen] = useState(resume !== undefined);
  const [step, setStep] = useState<Step>(
    resume === undefined ? { step: "kind" } : { step: "fields", kind: resume },
  );
  const openChange = (next: boolean) => {
    setOpen(next);
    if (next) {
      setStep({ step: "kind" });
    } else {
      forgetPlexPin();
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
