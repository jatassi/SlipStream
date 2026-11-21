import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";

import { statusQuery } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { Button } from "../ui/button";
import { AdminView } from "./admin-view";

export type Viewer = "admin" | "member" | "signed-out";

// A resume or a wipe ends Superseded, and the gate opens the page that fits.
function useLeaveOnceResumed() {
  const status = useQuery(statusQuery).data;
  const navigate = useNavigate();
  const left = status !== undefined && status.superseded !== true;
  useEffect(() => {
    if (left) {
      void navigate({ to: "/", replace: true });
    }
  }, [left, navigate]);
}

export function SupersededPage({ viewer }: { viewer: Viewer }) {
  useLeaveOnceResumed();
  return (
    <EntryPage title="This Controller was replaced">
      {viewer === "admin" ? <AdminView /> : null}
      {viewer === "member" ? <p>An Admin can wipe this Controller and join the new one.</p> : null}
      {viewer === "signed-out" ? (
        <>
          <p>Sign in as an Admin to wipe this Controller and join the new one.</p>
          <div>
            <Button look="primary" nativeButton={false} render={<Link to="/sign-in" />}>
              Sign in
            </Button>
          </div>
        </>
      ) : null}
    </EntryPage>
  );
}
