import { Chip } from "../ui/chip";

/** A Node's Roles as outlined chips, each its label and what it is doing, or "No Roles yet". */
export function RoleChips({
  roles,
}: {
  roles: readonly { readonly role: string; readonly words: string }[];
}) {
  return roles.length === 0 ? (
    <span className="text-meta text-ink-3">No Roles yet</span>
  ) : (
    <span className="flex flex-wrap gap-1.5">
      {roles.map(({ role, words }) => (
        <Chip key={role} look="outline">
          {words}
        </Chip>
      ))}
    </span>
  );
}
