package authors

import "errors"

var ErrNameRequired = errors.New("name is required")
var ErrAuthorNotFound = errors.New("author not found")
