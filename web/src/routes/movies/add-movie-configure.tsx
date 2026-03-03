import {
  FolderSelect,
  FormActions,
  ProfileSelect,
  ToggleField,
} from '@/components/media/media-configure-fields'
import { PosterImage } from '@/components/media/poster-image'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { MovieSearchResult, QualityProfile, RootFolder } from '@/types'

type AddMovieConfigureProps = {
  selectedMovie: MovieSearchResult
  rootFolderId: string
  setRootFolderId: (v: string) => void
  rootFolders: RootFolder[] | undefined
  qualityProfileId: string
  setQualityProfileId: (v: string) => void
  qualityProfiles: QualityProfile[] | undefined
  monitored: boolean
  setMonitored: (v: boolean) => void
  searchOnAdd: boolean | undefined
  setSearchOnAdd: (v: boolean) => void
  isPending: boolean
  handleBack: () => void
  handleAdd: () => void
}

export function AddMovieConfigure(props: AddMovieConfigureProps) {
  return (
    <div className="max-w-2xl space-y-6">
      <MoviePreview movie={props.selectedMovie} />
      <ConfigurationForm {...props} />
      <FormActions
        rootFolderId={props.rootFolderId}
        qualityProfileId={props.qualityProfileId}
        isPending={props.isPending}
        onBack={props.handleBack}
        onAdd={props.handleAdd}
        addLabel="Add Movie"
      />
    </div>
  )
}

function MoviePreview({ movie }: { movie: MovieSearchResult }) {
  return (
    <Card>
      <CardContent className="flex gap-4 p-4">
        <PosterImage
          url={movie.posterUrl}
          alt={movie.title}
          type="movie"
          className="h-36 w-24 shrink-0 rounded"
        />
        <div>
          <h2 className="text-xl font-semibold">{movie.title}</h2>
          <p className="text-muted-foreground">{movie.year ?? 'Unknown year'}</p>
          {movie.overview ? <p className="text-muted-foreground mt-2 line-clamp-3 text-sm">
              {movie.overview}
            </p> : null}
        </div>
      </CardContent>
    </Card>
  )
}

function ConfigurationForm(props: AddMovieConfigureProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <FolderSelect
          rootFolderId={props.rootFolderId}
          rootFolders={props.rootFolders}
          onChange={props.setRootFolderId}
        />
        <ProfileSelect
          qualityProfileId={props.qualityProfileId}
          qualityProfiles={props.qualityProfiles}
          onChange={props.setQualityProfileId}
        />
        <ToggleField
          label="Monitored"
          description="Automatically search for and download releases"
          checked={props.monitored}
          onChange={props.setMonitored}
        />
        <ToggleField
          label="Search on Add"
          description="Start searching for releases immediately"
          checked={props.searchOnAdd ?? false}
          onChange={props.setSearchOnAdd}
        />
      </CardContent>
    </Card>
  )
}
