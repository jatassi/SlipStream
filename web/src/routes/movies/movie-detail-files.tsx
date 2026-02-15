import { QualityBadge } from '@/components/media/quality-badge'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatBytes } from '@/lib/formatters'
import type { MovieFile, Slot } from '@/types'

type MovieDetailFilesProps = {
  files: MovieFile[]
  isMultiVersionEnabled: boolean
  expandedFileId: number | null
  enabledSlots: Slot[]
  isAssigning: boolean
  onToggleExpandFile: (fileId: number) => void
  onAssignFileToSlot: (fileId: number, slotId: number) => void
  getSlotName: (slotId: number | undefined) => string | null
}

export function MovieDetailFiles({
  files,
  isMultiVersionEnabled,
  expandedFileId,
  enabledSlots,
  isAssigning,
  onToggleExpandFile,
  onAssignFileToSlot,
  getSlotName,
}: MovieDetailFilesProps) {
  if (files.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Files</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">No files found</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Files</CardTitle>
      </CardHeader>
      <CardContent>
        <FilesTable
          files={files}
          expandedFileId={expandedFileId}
          isMultiVersionEnabled={isMultiVersionEnabled}
          enabledSlots={enabledSlots}
          isAssigning={isAssigning}
          onToggleExpandFile={onToggleExpandFile}
          onAssignFileToSlot={onAssignFileToSlot}
          getSlotName={getSlotName}
        />
      </CardContent>
    </Card>
  )
}

function FilesTable({
  files,
  expandedFileId,
  isMultiVersionEnabled,
  enabledSlots,
  isAssigning,
  onToggleExpandFile,
  onAssignFileToSlot,
  getSlotName,
}: Omit<MovieDetailFilesProps, 'files'> & { files: MovieFile[] }) {
  const showColumns = expandedFileId === null
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Filename</TableHead>
          {showColumns ? (
            <>
              <TableHead>Quality</TableHead>
              <TableHead>Video</TableHead>
              <TableHead>Audio</TableHead>
              {isMultiVersionEnabled ? <TableHead>Slot</TableHead> : null}
              <TableHead className="text-right">Size</TableHead>
            </>
          ) : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {files.map((file) => (
          <FileRow
            key={file.id}
            file={file}
            isExpanded={expandedFileId === file.id}
            showColumns={showColumns}
            isMultiVersionEnabled={isMultiVersionEnabled}
            enabledSlots={enabledSlots}
            isAssigning={isAssigning}
            onToggleExpand={() => onToggleExpandFile(file.id)}
            onAssignToSlot={(slotId) => onAssignFileToSlot(file.id, slotId)}
            getSlotName={getSlotName}
          />
        ))}
      </TableBody>
    </Table>
  )
}

type FileRowProps = {
  file: MovieFile
  isExpanded: boolean
  showColumns: boolean
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
  isAssigning: boolean
  onToggleExpand: () => void
  onAssignToSlot: (slotId: number) => void
  getSlotName: (slotId: number | undefined) => string | null
}

function FileRow({ file, isExpanded, showColumns, onToggleExpand, ...rest }: FileRowProps) {
  const filename = file.path.split('/').pop() ?? file.path
  const showDetails = !isExpanded && showColumns

  return (
    <TableRow>
      <TableCell className="cursor-pointer" onClick={onToggleExpand}>
        {isExpanded ? (
          <span className="font-mono text-xs break-all">{file.path}</span>
        ) : (
          <span className="font-mono text-sm">{filename}</span>
        )}
      </TableCell>
      {showDetails ? <FileDetailCells file={file} {...rest} /> : null}
    </TableRow>
  )
}

type FileDetailCellsProps = Pick<
  FileRowProps,
  'isMultiVersionEnabled' | 'enabledSlots' | 'isAssigning' | 'getSlotName'
> & {
  file: MovieFile
  onAssignToSlot: (slotId: number) => void
}

function FileDetailCells({
  file, isMultiVersionEnabled, enabledSlots, isAssigning, onAssignToSlot, getSlotName,
}: FileDetailCellsProps) {
  return (
    <>
      <TableCell>
        <QualityBadge quality={file.quality} />
      </TableCell>
      <VideoCell file={file} />
      <AudioCell file={file} />
      {isMultiVersionEnabled ? (
        <TableCell>
          <SlotSelect
            file={file}
            enabledSlots={enabledSlots}
            isAssigning={isAssigning}
            onAssign={onAssignToSlot}
            getSlotName={getSlotName}
          />
        </TableCell>
      ) : null}
      <TableCell className="text-right">{formatBytes(file.size)}</TableCell>
    </>
  )
}

function VideoCell({ file }: { file: MovieFile }) {
  return (
    <TableCell>
      <div className="flex items-center gap-1">
        {file.videoCodec ? (
          <Badge variant="outline" className="font-mono text-xs">{file.videoCodec}</Badge>
        ) : null}
        {file.dynamicRange?.split(' ').map((dr) => (
          <Badge key={dr} variant="outline" className="font-mono text-xs">{dr}</Badge>
        ))}
      </div>
    </TableCell>
  )
}

function AudioCell({ file }: { file: MovieFile }) {
  return (
    <TableCell>
      <div className="flex items-center gap-1">
        {file.audioCodec ? (
          <Badge variant="outline" className="font-mono text-xs">{file.audioCodec}</Badge>
        ) : null}
        {file.audioChannels ? (
          <Badge variant="outline" className="font-mono text-xs">{file.audioChannels}</Badge>
        ) : null}
      </div>
    </TableCell>
  )
}

function SlotSelect({
  file,
  enabledSlots,
  isAssigning,
  onAssign,
  getSlotName,
}: {
  file: MovieFile
  enabledSlots: Slot[]
  isAssigning: boolean
  onAssign: (slotId: number) => void
  getSlotName: (slotId: number | undefined) => string | null
}) {
  return (
    <Select
      value={file.slotId?.toString() ?? 'unassigned'}
      onValueChange={(value) => {
        if (value && value !== 'unassigned') {
          onAssign(Number.parseInt(value, 10))
        }
      }}
      disabled={isAssigning}
    >
      <SelectTrigger className="h-8 w-32">
        {getSlotName(file.slotId) ?? (
          <span className="text-muted-foreground">Unassigned</span>
        )}
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="unassigned" disabled>
          Unassigned
        </SelectItem>
        {enabledSlots.map((slot) => (
          <SelectItem key={slot.id} value={slot.id.toString()}>
            {slot.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
