export function Skeleton({ label }: { label: string }) {
  return (
    <output aria-label={label} className="flex flex-col gap-3">
      <span className="block h-8 w-48 rounded-md bg-fill" />
      <span className="block h-4 w-80 rounded-md bg-fill" />
      <span className="block h-4 w-64 rounded-md bg-fill" />
    </output>
  );
}
