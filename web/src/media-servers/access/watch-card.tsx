import { useSuspenseQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";

import { listed } from "../../targets/words";
import { Button } from "../../ui/button";
import { Section } from "../../ui/section";
import { myAccess } from "./portal";

export function WatchCard() {
  const pending = useSuspenseQuery(myAccess()).data.rows.filter((row) => row.pending);
  if (pending.length === 0) {
    return null;
  }
  return (
    <Section title="Watch">
      <div className="flex flex-col items-start gap-4 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset">
        <p className="text-ink-1">
          Your access to {listed(pending.map((row) => row.server.name))} isn&apos;t ready yet.
        </p>
        <Button
          nativeButton={false}
          render={
            <Link to="/welcome/$step" params={{ step: "watch" }} state={{ fromHome: true }} />
          }
        >
          See what&apos;s next
        </Button>
      </div>
    </Section>
  );
}
