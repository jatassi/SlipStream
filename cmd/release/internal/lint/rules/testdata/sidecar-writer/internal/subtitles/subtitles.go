package subtitles

import "example.com/app/internal/storing"

type Config struct{ Storage storing.SidecarRemote }

type Subtitles struct{ cfg Config }
