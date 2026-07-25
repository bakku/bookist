package books

import "context"

type Repository interface {
	List(ctx context.Context) ([]Book, error)
	Search(ctx context.Context, query string) ([]Book, error)
	ListByListID(ctx context.Context, listID int64) ([]Book, error)
	SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error)
	GetByID(ctx context.Context, id int64) (Book, error)
	Create(ctx context.Context, input CreateBookRequest) (Book, error)
	Update(ctx context.Context, id int64, input UpdateBookRequest) (Book, *string, error)
	Delete(ctx context.Context, id int64) error
}
