package contenttypetest

import "example.com/app/internal/contenttype"

var Fixture = contenttype.Define[struct{}]("fixture", nil, nil, nil)
