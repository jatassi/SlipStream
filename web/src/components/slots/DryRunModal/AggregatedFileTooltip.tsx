import type { AggregatedFileTooltipProps } from './types'

export function AggregatedFileTooltip({ files }: AggregatedFileTooltipProps) {
  const conflictFiles = files.filter((f) => f.conflict)
  const noMatchFiles = files.filter((f) => f.needsReview && !f.conflict)

  return (
    <div className="space-y-3">
      {conflictFiles.map((file) => (
        <div key={file.fileId} className="text-xs">
          <p className="mb-1 font-medium text-orange-600 dark:text-orange-400">{file.conflict}</p>
          {file.slotRejections && file.slotRejections.length > 0 ? (
            <div className="ml-2 space-y-1">
              {file.slotRejections.map((rejection) => (
                <div key={rejection.slotId}>
                  <span className="text-muted-foreground font-medium">{rejection.slotName}:</span>
                  <ul className="ml-1 list-inside list-disc">
                    {rejection.reasons.map((reason) => (
                      <li key={reason}>{reason}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      ))}
      {noMatchFiles.map((file) => (
        <div key={file.fileId} className="text-xs">
          <p className="mb-1 font-medium text-red-600 dark:text-red-400">No matching slot</p>
          {file.slotRejections && file.slotRejections.length > 0 ? (
            <div className="ml-2 space-y-1">
              {file.slotRejections.map((rejection) => (
                <div key={rejection.slotId}>
                  <span className="text-muted-foreground font-medium">{rejection.slotName}:</span>
                  <ul className="ml-1 list-inside list-disc">
                    {rejection.reasons.map((reason) => (
                      <li key={reason}>{reason}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  )
}
