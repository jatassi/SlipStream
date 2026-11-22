import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { useGetSupersededSuspense } from "../api/gen/query/superseded/superseded";
import { statusQuery } from "../session/session";
import { useClock } from "../ui/clock";
import { FenceDetails } from "./fence-details";
import { WipeForm } from "./wipe-form";
import { leadOf } from "./words";

const restartPoll = 2000;

/** What an Admin sees: the lead, Wipe and join, and the machinery under Details. */
export function AdminView() {
  const { data } = useGetSupersededSuspense();
  const clock = useClock();
  const [wiping, setWiping] = useState(false);
  // The wiped Node answers from a later process, as a Worker of the new
  // Controller or Unclaimed, and the page leaves once status says so.
  useQuery({ ...statusQuery, enabled: wiping, refetchInterval: restartPoll, retry: false });
  if (wiping) {
    return <p>Wiping this Controller. It restarts, then joins the new Controller as a Worker.</p>;
  }
  return (
    <>
      <p className="text-ink-1">{leadOf(data, clock)}</p>
      <WipeForm
        onWiping={() => {
          setWiping(true);
        }}
      />
      <FenceDetails view={data} />
    </>
  );
}
