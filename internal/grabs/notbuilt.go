package grabs

import "errors"

// ErrNotBuilt is an operation of S32 (Manual search and grab) whose body a
// later change of that build writes. The API answers it 501. The last
// change deletes this file.
var ErrNotBuilt = errors.New("this operation isn't built yet")
