package library

import "example.com/app/internal/contenttype"

var Books = contenttype.Define[struct{}]("books", nil, nil, nil)
