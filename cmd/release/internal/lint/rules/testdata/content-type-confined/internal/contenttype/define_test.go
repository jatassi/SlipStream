package contenttype_test

import "example.com/app/internal/contenttype"

var bad = contenttype.Define[struct{}]("Fixture", nil, nil, nil)
