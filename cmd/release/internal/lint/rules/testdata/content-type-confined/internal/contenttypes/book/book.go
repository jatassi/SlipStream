package book

import "example.com/app/internal/contenttype"

const id = "book"

var Def = contenttype.Define[struct{}](id, nil, nil, nil)
