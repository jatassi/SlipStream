import { cn } from "@/lib/utils"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("bg-muted rounded-md animate-pulse relative overflow-hidden", className)}
      {...props}
    >
      <div className="absolute inset-y-0 left-0 w-[200%] animate-skeleton-shimmer bg-[linear-gradient(90deg,transparent_0%,transparent_25%,rgba(255,255,255,0.03)_35%,rgba(255,255,255,0.10)_50%,rgba(255,255,255,0.03)_65%,transparent_75%,transparent_100%)]" />
    </div>
  )
}

export { Skeleton }
