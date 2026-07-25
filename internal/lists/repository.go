package lists

import "context"

type Repository interface {
	Create(ctx context.Context, input CreateListRequest) (List, error)
	List(ctx context.Context) ([]List, error)
	Search(ctx context.Context, query string) ([]List, error)
	NameExists(ctx context.Context, name string) (bool, error)
	GetByID(ctx context.Context, id int64) (List, error)
	Delete(ctx context.Context, id int64) error
	AddBookToList(ctx context.Context, listID, bookID int64) error
	RemoveBookFromList(ctx context.Context, listID, bookID int64) error
}
