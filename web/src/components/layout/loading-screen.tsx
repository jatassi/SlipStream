import { Loader2 } from 'lucide-react'

export function LoadingScreen() {
  return (
    <div className="bg-background flex h-full min-h-48 items-center justify-center">
      <Loader2 className="text-muted-foreground size-8 animate-spin" />
    </div>
  )
}
