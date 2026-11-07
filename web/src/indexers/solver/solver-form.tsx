import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import type { IndexersChallengeSolver, IndexersSolverChange, NodesNode } from "../../api/gen/model";
import { setChallengeSolver } from "../../api/gen/query/indexers/indexers";
import { adoptChallengeSolver } from "../../live/adopt";
import { Button } from "../../ui/button";
import { useAppForm } from "../../ui/form";
import { toast } from "../../ui/toast";
import { refusal, solverSchema, valuesOf } from "./values";

function useSave(node: NodesNode) {
  const queryClient = useQueryClient();
  return async (change: IndexersSolverChange) => {
    adoptChallengeSolver(queryClient, node.id, await setChallengeSolver(node.id, change));
  };
}

function useSolverForm(node: NodesNode, solver: IndexersChallengeSolver) {
  const save = useSave(node);
  return useAppForm({
    defaultValues: valuesOf(solver),
    validators: { onSubmit: solverSchema },
    onSubmit: async ({ value, formApi }) => {
      const next = solverSchema.parse(value);
      try {
        await save({ address: next.address, timeout: Number(next.timeout) });
      } catch (error) {
        formApi.setErrorMap({ onSubmit: refusal(error) });
        return;
      }
      toast(`Saved ${node.name}'s Challenge solver`);
    },
  });
}

function RemoveSolver({ node }: { node: NodesNode }) {
  const save = useSave(node);
  const [busy, setBusy] = useState(false);
  const remove = async () => {
    setBusy(true);
    try {
      await save({ address: "" });
      toast(`Removed ${node.name}'s Challenge solver`);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setBusy(false);
    }
  };
  return (
    <Button busy={busy} onClick={() => void remove()}>
      Remove
    </Button>
  );
}

/** The solver's address and timeout, saved once the Node reaches the address. */
export function SolverForm({ node, solver }: { node: NodesNode; solver: IndexersChallengeSolver }) {
  const form = useSolverForm(node, solver);
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.AppField name="address">
          {(field) => (
            <field.TextField
              label="Address"
              type="url"
              autoComplete="off"
              placeholder="http://localhost:8191"
              data
              description={`A FlareSolverr or Byparr service that ${node.name} reaches.`}
            />
          )}
        </form.AppField>
        <div className="w-40">
          <form.AppField name="timeout">
            {(field) => (
              <field.TextField
                label="Timeout"
                inputMode="numeric"
                description="Seconds, from 10 to 180."
              />
            )}
          </form.AppField>
        </div>
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          Saving tests the solver from {node.name} first.
        </p>
        <form.FormAlert />
        <div className="flex flex-wrap justify-end gap-2">
          {solver.address !== null && <RemoveSolver node={node} />}
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}
