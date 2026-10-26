import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";

import type { FirstrunStepState } from "../../api/gen/model";
import { setFirstRunStep } from "../../api/gen/query/firstrun/firstrun";
import { type Place, around } from "../../session/first-run";
import { adoptFirstRun, finishFirstSetup } from "../../session/session";
import { EntryPage } from "../../shell/entry-page";
import { DoneScreen } from "./done-screen";
import { FinishSetup, Progress, type StepNav } from "./frame";
import { screens } from "./screens";

function useGo() {
  const navigate = useNavigate();
  return (to: Place) => navigate({ to: "/setup/$step", params: { step: to } });
}

/** Finishes first setup, then goes to `href` in the app. */
function useLeave() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  return async (href: string) => {
    await finishFirstSetup(queryClient);
    await navigate({ href });
  };
}

function useNav(step: FirstrunStepState, next: Place, back: Place | undefined): StepNav {
  const queryClient = useQueryClient();
  const go = useGo();
  const store = async (outcome: "done" | "skipped") => {
    adoptFirstRun(queryClient, await setFirstRunStep(step.id, { outcome }));
    await go(next);
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

export function StepPage({
  step,
  shown,
}: {
  step: FirstrunStepState;
  shown: readonly FirstrunStepState[];
}) {
  const screen = screens[step.id];
  const { number, count, back, next, ahead } = around(shown, step.id);
  const nav = useNav(step, next, back);
  const leave = useLeave();
  return (
    <EntryPage title={screen.title}>
      <Progress number={number ?? 0} count={count} ahead={ahead.map((id) => screens[id].name)} />
      <p>{screen.why}</p>
      <screen.Body key={step.id} step={step} nav={nav} />
      <FinishSetup finish={() => leave("/library")} />
    </EntryPage>
  );
}

export function DonePage({ shown }: { shown: readonly FirstrunStepState[] }) {
  const go = useGo();
  const { back } = around(shown, "done");
  const leave = useLeave();
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
