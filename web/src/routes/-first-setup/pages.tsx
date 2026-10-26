import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import type { FirstrunStepState } from "../../api/gen/model";
import { type Place, around } from "../../session/first-run";
import { storeStep } from "../../session/session";
import { EntryPage } from "../../shell/entry-page";
import { DoneScreen } from "./done-screen";
import { FinishSetup, Progress, type StepNav } from "./frame";
import { screens } from "./screens";

function useGo() {
  const navigate = useNavigate();
  return (to: Place) => navigate({ to: "/setup/$step", params: { step: to } });
}

function useNav(step: FirstrunStepState, next: Place, back: Place | undefined): StepNav {
  const queryClient = useQueryClient();
  const go = useGo();
  const store = async (outcome: "done" | "skipped") => {
    if (await storeStep(queryClient, step.id, outcome)) {
      await go(next);
    }
  };
  return {
    complete: async () => {
      await (step.state === "done" ? go(next) : store("done"));
    },
    skip: () => store("skipped"),
    back:
      back === undefined
        ? undefined
        : () => {
            void go(back);
          },
  };
}

type Leave = (href: string) => Promise<void>;

export function StepPage({
  step,
  shown,
  leave,
}: {
  step: FirstrunStepState;
  shown: readonly FirstrunStepState[];
  leave: Leave;
}) {
  const screen = screens[step.id];
  const { number, count, back, next, ahead } = around(shown, step.id);
  const nav = useNav(step, next, back);
  return (
    <EntryPage title={screen.title}>
      <Progress number={number ?? 0} count={count} ahead={ahead.map((id) => screens[id].name)} />
      <p>{screen.why}</p>
      <screen.Body key={step.id} step={step} nav={nav} />
      <FinishSetup finish={() => leave("/library")} />
    </EntryPage>
  );
}

export function DonePage({ shown, leave }: { shown: readonly FirstrunStepState[]; leave: Leave }) {
  const go = useGo();
  const { back } = around(shown, "done");
  return (
    <EntryPage title="First setup">
      <DoneScreen
        shown={shown}
        back={
          back === undefined
            ? undefined
            : () => {
                void go(back);
              }
        }
        finish={leave}
      />
    </EntryPage>
  );
}
