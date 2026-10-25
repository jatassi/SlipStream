import { Link, type ErrorComponentProps, useRouter } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { UnreachableError, isApiError, isProblem } from "../api/errors";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { type NavRow, adminNav } from "./nav";
import { thisPage } from "./reload";

export type Fact = { readonly label: string; readonly value: ReactNode };

type Failure = {
  readonly sentence: string;
  readonly facts: readonly Fact[];
  readonly forAdmins: boolean;
};

function failureOf(error: unknown): Failure {
  if (isApiError(error)) {
    const forAdmins = isProblem(error, "needs_admin");
    return {
      sentence: forAdmins ? "This page is for Admins." : error.title,
      facts: [
        { label: "Code", value: <Data>{error.code}</Data> },
        { label: "Status", value: <Data>{error.status}</Data> },
        ...(error.detail === undefined ? [] : [{ label: "Detail", value: error.detail }]),
      ],
      forAdmins,
    };
  }
  return {
    sentence:
      error instanceof UnreachableError
        ? "SlipStream isn't responding. Check that it's running, then try again."
        : "This page couldn't load.",
    facts: [{ label: "Message", value: error instanceof Error ? error.message : String(error) }],
    forAdmins: false,
  };
}

const sentence = "text-sentence text-ink-1 in-data-[density=touch]:text-sentence-touch";

const card =
  "flex max-w-[560px] flex-col gap-2 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset";

/** The error page's card: one plain sentence, its action, then the machinery under Details. */
export function ProblemCard({
  said,
  action,
  facts,
}: {
  said: string;
  action: ReactNode;
  facts: readonly Fact[];
}) {
  return (
    <div className={card}>
      <Details
        summary={
          <span role="alert" className={sentence}>
            {said}
          </span>
        }
        action={<div className="my-2">{action}</div>}
      >
        <dl className="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
          {facts.map(({ label, value }) => (
            <div key={label} className="contents">
              <dt className="text-ink-3">{label}</dt>
              <dd className="wrap-anywhere">{value}</dd>
            </div>
          ))}
        </dl>
      </Details>
    </div>
  );
}

function FailureCard({ error, retry }: { error: unknown; retry: ReactNode }) {
  const { sentence: said, facts, forAdmins } = failureOf(error);
  return (
    <ProblemCard
      said={said}
      facts={facts}
      action={
        forAdmins ? (
          <Button nativeButton={false} render={<Link to="/portal" />}>
            Open the portal
          </Button>
        ) : (
          retry
        )
      }
    />
  );
}

export function RouteError({ error, reset }: ErrorComponentProps) {
  const router = useRouter();
  return (
    <FailureCard
      error={error}
      retry={
        <Button
          onClick={() => {
            reset();
            void router.invalidate();
          }}
        >
          Try again
        </Button>
      }
    />
  );
}

function FullScreen({ children }: { children: ReactNode }) {
  return <div className="grid min-h-dvh place-items-center p-4">{children}</div>;
}

/** The shell itself failed, so the card takes the screen and a reload rebuilds it. */
export function ShellError({ error }: ErrorComponentProps) {
  return (
    <FullScreen>
      <FailureCard
        error={error}
        retry={
          <Button
            onClick={() => {
              thisPage.reload();
            }}
          >
            Reload
          </Button>
        }
      />
    </FullScreen>
  );
}

export function NotFound({ home }: { home: NavRow }) {
  return (
    <div className={card}>
      <p className={sentence}>There&apos;s nothing at this address.</p>
      <div className="mt-2">
        <Button nativeButton={false} render={<Link to={home.to} />}>
          Go to {home.label}
        </Button>
      </div>
    </div>
  );
}

// An address outside both shells has no audience yet; Library's guard sends a
// Member on to the portal.
export function RootNotFound() {
  return (
    <FullScreen>
      <NotFound home={adminNav[0]} />
    </FullScreen>
  );
}
