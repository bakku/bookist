package authors

import "errors"

var ErrNameRequired = errors.New("name is required")
var ErrNoFieldsToUpdate = errors.New("no fields to update")
var ErrAuthorNotFound = errors.New("author not found")
