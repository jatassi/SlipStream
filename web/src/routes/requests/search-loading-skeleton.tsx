export function SearchLoadingSkeleton() {
  return (
    <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
      {Array.from({ length: 12 }, (_, i) => (
        <div key={i} className="bg-muted aspect-[2/3] animate-pulse rounded-lg" />
      ))}
    </div>
  )
}
