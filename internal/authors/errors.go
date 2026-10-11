package authors

import "bakku.dev/bookist/internal/apperr"

var ErrNameRequired = apperr.Invalid("name_required", "name is required", "name")
var ErrNoFieldsToUpdate = apperr.Invalid("no_fields_to_update", "no fields to update", "")
var ErrAuthorNotFound = apperr.NotFound("author_not_found", "author not found")
