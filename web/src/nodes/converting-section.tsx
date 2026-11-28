import { Link } from "@tanstack/react-router";

import type { NodesNode } from "../api/gen/model";
import { useGetNodeConvertingSuspense } from "../api/gen/query/conversions/conversions";
import { useListTasksSuspense } from "../api/gen/query/tasks/tasks";
import { bytes } from "../format";
import { Callout } from "../ui/callout";
import { ConfigFields } from "./converting-config";
import { ConvertingProofs, convertsWith } from "./converting-proofs";
import { RerunButton } from "./converting-rerun";
import { ConvertingSettings } from "./converting-settings";

/**
 * A Node's Converting section: what it converts with, its settings, its
 * scratch folder and ffmpeg path while it is online, and its test runs.
 */
export function ConvertingSection({ node }: { node: NodesNode }) {
  const converting = useGetNodeConvertingSuspense(node.id).data;
  const { zone } = useListTasksSuspense().data.settings.window;
  const { running, scratch, override, proofs } = converting;
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <p className="text-ink-1">{convertsWith(node.name, converting)}</p>
        {running > 0 && (
          <p className="text-ink-2">
            <Link
              to="/activity/conversions"
              search={{ node: node.id }}
              className="underline decoration-ink-3 underline-offset-4 hover:decoration-ink-1"
            >
              {running === 1 ? "1 Conversion" : `${String(running)} Conversions`}
            </Link>{" "}
            {running === 1 ? "is" : "are"} running on {node.name} now.
          </p>
        )}
      </div>
      {override && (
        <Callout tone="warn" chip="Can't run">
          {`A Target runs its Conversions only on ${node.name}, which can't run them. Change Run on in that Target's Conversion settings.`}
        </Callout>
      )}
      <ConvertingSettings nodeId={node.id} converting={converting} zone={zone} />
      {node.standing === "in-step" && (
        <ConfigFields
          node={node}
          free={
            scratch === null ? undefined : `${bytes(scratch.free)} free of ${bytes(scratch.total)}`
          }
        />
      )}
      <div className="flex flex-col items-start gap-2">
        <ConvertingProofs proofs={proofs} />
        <RerunButton node={node} />
      </div>
    </div>
  );
}
