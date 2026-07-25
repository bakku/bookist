package lists

import "errors"

var ErrNameRequired = errors.New("name is required")
var ErrNameConflict = errors.New("a list with this name already exists")
var ErrListNotFound = errors.New("list not found")
var ErrBookNotFound = errors.New("book not found")
var ErrBookAlreadyInList = errors.New("book is already in this list")
var ErrBookNotInList = errors.New("book is not in this list")
