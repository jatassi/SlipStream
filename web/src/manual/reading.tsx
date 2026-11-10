import { useQuery } from "@tanstack/react-query";
import { type ReactNode, createContext, useMemo } from "react";

import { getListTargetsQueryKey, listTargets } from "../api/gen/query/targets/targets";
import { useTester } from "../formats/tester-dialog";
import { useNow } from "../tasks/use-now";

/** What the sheet's lines read beyond the search: the time, Target names and the tester. */
type Reading = {
  readonly now: Date;
  readonly targets: ReadonlyMap<string, string>;
  readonly onTest: (title: string) => void;
};

export const ReadingContext = createContext<Reading>({
  now: new Date(0),
  targets: new Map(),
  onTest: () => {},
});

const minute = 60_000;

/** A Release's Open in tester judges it against the Work the sheet searches. */
export function ReadingProvider({
  workId,
  title,
  children,
}: {
  workId: string;
  title: string;
  children: ReactNode;
}) {
  const now = useNow(minute);
  const tester = useTester();
  const targets = useQuery({
    queryKey: getListTargetsQueryKey(),
    queryFn: ({ signal }) => listTargets({ signal }),
  }).data?.targets;
  const { open } = tester;
  const reading = useMemo(
    () => ({
      now,
      targets: new Map(targets?.map(({ id, name }) => [id, name])),
      onTest: (name: string) => {
        open({ names: [name], work: { id: workId, label: title } });
      },
    }),
    [now, targets, open, workId, title],
  );
  return (
    <ReadingContext value={reading}>
      {children}
      {tester.dialog}
    </ReadingContext>
  );
}
