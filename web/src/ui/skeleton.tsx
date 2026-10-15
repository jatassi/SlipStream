export function Skeleton({ label }: { label: string }) {
  return (
    <output aria-label={label} className="flex flex-col gap-3">
      <span className="block h-8 w-48 rounded-item bg-fill" />
      <span className="block h-4 w-80 max-w-full rounded-chip bg-fill" />
      <span className="block h-4 w-64 max-w-full rounded-chip bg-fill" />
    </output>
  );
}
