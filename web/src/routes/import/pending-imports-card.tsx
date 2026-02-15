import { Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useGlobalLoading } from '@/hooks'
import { usePendingImports, useRetryImport } from '@/hooks/use-import'

function PendingImportsSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-4 w-48" />
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {Array.from({ length: 3 }, (_, i) => (
            <div key={i} className="flex items-center justify-between rounded-lg border p-2">
              <div className="min-w-0 flex-1 space-y-1.5">
                <Skeleton className="h-4 w-48" />
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
              <Skeleton className="h-8 w-14 rounded-md" />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export function PendingImportsCard() {
  const globalLoading = useGlobalLoading()
  const { data: pending, isLoading: queryLoading } = usePendingImports()
  const retryMutation = useRetryImport()
  const isLoading = queryLoading || globalLoading

  if (isLoading) {return <PendingImportsSkeleton />}
  if (!pending || pending.length === 0) {return null}

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Pending Imports</CardTitle>
        <CardDescription>Files waiting to be imported</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {pending.map((item) => (
            <div
              key={item.id ?? item.filePath}
              className="flex items-center justify-between rounded-lg border p-2"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{item.fileName}</p>
                <div className="mt-1 flex items-center gap-2">
                  <Badge variant={item.status === 'failed' ? 'destructive' : 'outline'}>
                    {item.status}
                  </Badge>
                  {item.isProcessing ? <Loader2 className="size-3 animate-spin" /> : null}
                </div>
                {item.error ? <p className="mt-1 text-xs text-red-600">{item.error}</p> : null}
              </div>
              {item.status === 'failed' && item.id ? (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => item.id && retryMutation.mutate(item.id)}
                  disabled={retryMutation.isPending}
                >
                  Retry
                </Button>
              ) : null}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
