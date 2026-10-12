package lists

import "bakku.dev/bookist/internal/apperr"

var ErrNameRequired = apperr.Invalid("name_required", "name is required", "name")
var ErrNoFieldsToUpdate = apperr.Invalid("no_fields_to_update", "no fields to update", "")
var ErrNameConflict = apperr.Conflict("list_name_conflict", "a list with this name already exists", "name")
var ErrBookAlreadyInList = apperr.Conflict("book_already_in_list", "book is already in this list", "book_id")
var ErrBookNotInList = apperr.NotFound("book_not_in_list", "book is not in this list")
