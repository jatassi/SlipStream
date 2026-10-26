import { Link } from "@tanstack/react-router";

import { sentenceOf } from "../api/errors";
import type { WorksContentType } from "../api/gen/model";
import { makeDefaultRoot, useListRootsSuspense } from "../api/gen/query/roots/roots";
import { makeDefaultProfile, useListProfilesSuspense } from "../api/gen/query/targets/targets";
import { Select } from "../ui/select";
import { toast } from "../ui/toast";

function Missing({
  type,
  what,
  section,
}: {
  type: WorksContentType;
  what: string;
  section: { id: string; label: string };
}) {
  return (
    <p className="text-ink-2">
      {type.name} has no {what} yet. Add one in{" "}
      <Link
        to="/settings/$section"
        params={{ section: section.id }}
        className="text-ink-1 underline"
      >
        {section.label}
      </Link>
      .
    </p>
  );
}

async function making(make: () => Promise<unknown>, done: string) {
  try {
    await make();
    toast(done);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}

export function DefaultProfile({ type }: { type: WorksContentType }) {
  const profiles = useListProfilesSuspense().data.profiles.filter(
    (profile) => profile.contentType === type.id,
  );
  const chosen = profiles.find((profile) => profile.default);
  if (chosen === undefined) {
    return <Missing type={type} what="Profile" section={{ id: "profiles", label: "Profiles" }} />;
  }
  return (
    <Select
      label="Default Profile"
      options={profiles.map((profile) => ({ value: profile.id, label: profile.name }))}
      value={chosen.id}
      onValueChange={(id) => {
        const name = profiles.find((profile) => profile.id === id)?.name;
        void making(() => makeDefaultProfile(id), `New ${type.work.many} now take ${name}.`);
      }}
    />
  );
}

export function DefaultRoot({ type }: { type: WorksContentType }) {
  const roots = useListRootsSuspense().data.roots.filter((root) => root.contentType === type.id);
  const chosen = roots.find((root) => root.default);
  if (chosen === undefined) {
    return (
      <Missing type={type} what="root" section={{ id: "library-roots", label: "Library roots" }} />
    );
  }
  return (
    <Select
      label="Default root"
      options={roots.map((root) => ({ value: root.id, label: root.path }))}
      value={chosen.id}
      onValueChange={(id) => {
        const path = roots.find((root) => root.id === id)?.path;
        void making(() => makeDefaultRoot(id), `New ${type.work.many} now go into ${path}.`);
      }}
    />
  );
}
