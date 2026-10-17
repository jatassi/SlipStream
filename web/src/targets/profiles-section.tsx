import { Add01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { TargetsProfile, WorksContentType } from "../api/gen/model";
import { useListProfilesSuspense, useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { Chip } from "../ui/chip";
import { Section } from "../ui/section";
import { type Column, Table } from "../ui/table";
import { ProfileMenu } from "./profile-actions";

type Group = { readonly id: string; readonly label: string; readonly profiles: TargetsProfile[] };

/** The Profiles by Content Type, in the order the types are listed; a type with none is left out. */
export function groupsOf(
  profiles: readonly TargetsProfile[],
  types: readonly WorksContentType[],
): Group[] {
  const ids = [
    ...new Set([...types.map((type) => type.id), ...profiles.map((p) => p.contentType)]),
  ];
  return ids.flatMap((id) => {
    const held = profiles.filter((profile) => profile.contentType === id);
    const label = types.find((type) => type.id === id)?.work.many ?? id;
    return held.length === 0 ? [] : [{ id, label, profiles: held }];
  });
}

const worksOf = (count: number) => (count === 1 ? "1 Work" : `${String(count)} Works`);

function columnsOf(
  label: string,
  targetName: (id: string) => string,
): readonly [Column<TargetsProfile>, ...Column<TargetsProfile>[]] {
  return [
    {
      id: "name",
      header: "Name",
      cell: (profile) => (
        <span className="flex flex-wrap items-center gap-2">
          <Link
            to="/settings/profiles/$profileId"
            params={{ profileId: profile.id }}
            className="font-medium hover:underline"
          >
            {profile.name}
          </Link>
          {profile.default && <Chip>Default</Chip>}
        </span>
      ),
    },
    {
      id: "targets",
      header: "Targets",
      cell: (profile) =>
        profile.targets.length === 0
          ? "None"
          : profile.targets.map((entry) => targetName(entry.targetId)).join(", "),
    },
    { id: "works", header: "Works", cell: (profile) => worksOf(profile.works) },
    {
      id: "actions",
      header: "Actions",
      align: "end",
      cell: (profile) => <ProfileMenu profile={profile} works={label} />,
    },
  ];
}

export function ProfilesSection() {
  const { profiles } = useListProfilesSuspense().data;
  const { targets } = useListTargetsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  const targetName = (id: string) => targets.find((target) => target.id === id)?.name ?? id;
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-ink-2">
          A Profile is the Targets a Work wants, in order, and the cuts it wants them in. New Works
          take their Content Type&apos;s default.
        </p>
        <Button nativeButton={false} icon={Add01Icon} render={<Link to="/settings/profiles/new" />}>
          New Profile
        </Button>
      </div>
      {groupsOf(profiles, contentTypes).map((group) => (
        <Section key={group.id} title={group.label} count={group.profiles.length}>
          <Table
            label={`${group.label} Profiles`}
            columns={columnsOf(group.label, targetName)}
            rows={group.profiles}
            rowKey={(profile) => profile.id}
            stackBelow={560}
          />
        </Section>
      ))}
    </div>
  );
}
