import type { AggregatedFileTooltipProps } from './types'

export function AggregatedFileTooltip({ files }: AggregatedFileTooltipProps) {
  const conflictFiles = files.filter(f => f.conflict)
  const noMatchFiles = files.filter(f => f.needsReview && !f.conflict)

  return (
    <div className="space-y-3">
      {conflictFiles.map((file) => (
        <div key={file.fileId} className="text-xs">
          <p className="font-medium text-orange-600 dark:text-orange-400 mb-1">{file.conflict}</p>
          {file.slotRejections && file.slotRejections.length > 0 && (
            <div className="space-y-1 ml-2">
              {file.slotRejections.map((rejection) => (
                <div key={rejection.slotId}>
                  <span className="font-medium text-muted-foreground">{rejection.slotName}:</span>
                  <ul className="list-disc list-inside ml-1">
                    {rejection.reasons.map((reason, idx) => (
                      <li key={idx}>{reason}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </div>
      ))}
      {noMatchFiles.map((file) => (
        <div key={file.fileId} className="text-xs">
          <p className="font-medium text-red-600 dark:text-red-400 mb-1">No matching slot</p>
          {file.slotRejections && file.slotRejections.length > 0 && (
            <div className="space-y-1 ml-2">
              {file.slotRejections.map((rejection) => (
                <div key={rejection.slotId}>
                  <span className="font-medium text-muted-foreground">{rejection.slotName}:</span>
                  <ul className="list-disc list-inside ml-1">
                    {rejection.reasons.map((reason, idx) => (
                      <li key={idx}>{reason}</li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
