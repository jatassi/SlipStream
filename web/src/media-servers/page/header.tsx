import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import type {
  MediaserversServerView as Server,
  MediaserversTested as Tested,
} from "../../api/gen/model";
import {
  confirmMediaServerIdentity,
  testMediaServer,
} from "../../api/gen/query/mediaservers/mediaservers";
import { adoptMediaServer } from "../../live/adopt-media-servers";
import { Button } from "../../ui/button";
import { toast } from "../../ui/toast";
import { SignalLine } from "../lines";

type TestAnswer = { readonly tested: Tested } | { readonly refused: string };

const fallbackWords: { readonly [F in Tested["fallback"]]: string } = {
  available: "The Controller sends changes while the reaching Node is offline.",
  none: "Changes wait while the reaching Node is offline, because the Controller can't reach the server.",
  untested: "The Controller's route wasn't tested.",
};

function TestResult({ answer }: { answer: TestAnswer }) {
  if ("refused" in answer) {
    return (
      <div role="alert">
        <SignalLine level="fail" text={answer.refused} />
      </div>
    );
  }
  const { reaching, controller, fallback } = answer.tested;
  return (
    <output className="flex flex-col gap-1.5">
      <SignalLine level={reaching.ok ? "ok" : "fail"} text={reaching.words} />
      <SignalLine level={controller.ok ? "ok" : "warn"} text={controller.words} />
      <SignalLine level={fallback === "available" ? "ok" : "warn"} text={fallbackWords[fallback]} />
    </output>
  );
}

/** What the server needs, then Test, which runs both routes, and Confirm on a new server Id. */
export function ServerHeader({ server }: { server: Server }) {
  const queryClient = useQueryClient();
  const [answer, setAnswer] = useState<TestAnswer>();
  const [testing, setTesting] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const test = async () => {
    setTesting(true);
    setAnswer(undefined);
    try {
      setAnswer({ tested: await testMediaServer(server.id) });
    } catch (error) {
      setAnswer({ refused: sentenceOf(error) });
    } finally {
      setTesting(false);
    }
  };
  const confirm = async () => {
    setConfirming(true);
    try {
      adoptMediaServer(queryClient, await confirmMediaServerIdentity(server.id));
      toast(`Confirmed the new server at ${server.address}`);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setConfirming(false);
    }
  };
  return (
    <div className="mb-6 flex flex-col gap-3">
      <SignalLine level={server.state.level} text={server.state.words} />
      <div className="flex flex-wrap gap-2">
        <Button size="small" busy={testing} onClick={() => void test()}>
          Test
        </Button>
        {server.contact === "other_server" && (
          <Button size="small" look="warn" busy={confirming} onClick={() => void confirm()}>
            Confirm the new server
          </Button>
        )}
      </div>
      {answer !== undefined && <TestResult answer={answer} />}
    </div>
  );
}
