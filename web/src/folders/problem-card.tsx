import type { ReactNode } from "react";

import type { RootsFolderProblemKind } from "../api/gen/model";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import type { FolderProblem } from "./use-folder-listing";

const unreadableWords: { readonly [K in RootsFolderProblemKind]: (node: string) => string } = {
  not_found: () => "This folder isn't there.",
  not_a_folder: () => "This is a file, not a folder.",
  denied: (node) => `SlipStream on ${node} isn't allowed to open this folder.`,
  unresponsive: (node) =>
    `${node} hasn't answered about this folder. A network share or a sleeping disk can be slow.`,
  io: (node) => `${node} couldn't read this folder.`,
};

function wordsOf(problem: FolderProblem, node: string): { sentence: string; details?: ReactNode } {
  switch (problem.kind) {
    case "unreadable": {
      const { kind, osText, account } = problem.problem;
      return {
        sentence: unreadableWords[kind](node),
        details: (
          <div className="flex flex-col gap-1">
            {kind === "denied" && account !== undefined && (
              <p>
                SlipStream runs as {account} on {node}. Give that account read access, or pick
                another folder.
              </p>
            )}
            <Data className="wrap-anywhere">{osText}</Data>
          </div>
        ),
      };
    }
    case "offline":
      return { sentence: `${node} is offline. Its folders can't be browsed until it reconnects.` };
    case "refused":
      return { sentence: problem.sentence };
    default:
      return problem satisfies never;
  }
}

// A stable callback, so the button takes focus once, as the card appears.
const focusOnMount = (element: HTMLElement | null) => {
  element?.focus();
};

export function ProblemCard({
  problem,
  nodeName,
  onUp,
  onRetry,
}: {
  problem: FolderProblem;
  nodeName: string;
  /** Absent on Places, which has nothing above it. */
  onUp: (() => void) | undefined;
  onRetry: () => void;
}) {
  const { sentence, details } = wordsOf(problem, nodeName);
  const actions = (
    <div className="flex gap-2 pt-2">
      {onUp !== undefined && (
        <Button ref={focusOnMount} onClick={onUp}>
          Go up
        </Button>
      )}
      <Button ref={onUp === undefined ? focusOnMount : undefined} onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
  return (
    <div className="m-3 rounded-card bg-glass-raised p-4 ring-1 ring-hairline ring-inset">
      {details === undefined ? (
        <div className="text-body text-ink-1 in-data-[density=touch]:text-body-touch">
          <p>{sentence}</p>
          {actions}
        </div>
      ) : (
        <Details summary={sentence} action={actions}>
          {details}
        </Details>
      )}
    </div>
  );
}
