package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	data := struct {
		Books []books.Book
	}{Books: bookList}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleAPIListBooks(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bookList)
}

func (s *Server) handleAPICreateBook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input books.CreateBookRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	book, err := s.books.Create(r.Context(), input)
	if err != nil {
		writeCreateBookError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, book)
}

func (s *Server) handleAPIListAuthors(w http.ResponseWriter, r *http.Request) {
	authorList, err := s.authors.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list authors", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, authorList)
}

func (s *Server) handleAPICreateAuthor(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input authors.CreateAuthorRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	author, err := s.authors.Create(r.Context(), input)
	if err != nil {
		writeCreateAuthorError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, author)
}

func writeCreateBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, books.ErrTitleRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errors.Is(err, books.ErrAuthorNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errors.Is(err, books.ErrInvalidFormat) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Error(w, "failed to create book", http.StatusInternalServerError)
}

func writeCreateAuthorError(w http.ResponseWriter, err error) {
	if errors.Is(err, authors.ErrNameRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Error(w, "failed to create author", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) handleAPIListLists(w http.ResponseWriter, r *http.Request) {
	listList, err := s.lists.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list lists", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, listList)
}

func (s *Server) handleAPICreateList(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input lists.CreateListRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	list, err := s.lists.Create(r.Context(), input)
	if err != nil {
		writeCreateListError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, list)
}

func (s *Server) handleAPIListBooksInList(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("id")

	_, err := s.lists.GetByID(r.Context(), listID)
	if err != nil {
		if errors.Is(err, lists.ErrListNotFound) {
			http.Error(w, "list not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get list", http.StatusInternalServerError)
		return
	}

	listBooks, err := s.lists.ListBooks(r.Context(), listID)
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	if len(listBooks) > 0 {
		bookIDs := make([]string, len(listBooks))
		for i, b := range listBooks {
			bookIDs[i] = b.ID
		}
		authorsByBook, err := s.authors.ListByBookIDs(r.Context(), bookIDs)
		if err != nil {
			http.Error(w, "failed to hydrate authors", http.StatusInternalServerError)
			return
		}
		for i, b := range listBooks {
			if aa, ok := authorsByBook[b.ID]; ok {
				listBooks[i].Authors = aa
			}
		}
	}

	writeJSON(w, http.StatusOK, listBooks)
}

func (s *Server) handleAPIAddBookToList(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	listID := r.PathValue("id")

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input lists.AddBookToListRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(input.BookID) == "" {
		http.Error(w, "book_id is required", http.StatusBadRequest)
		return
	}

	err := s.lists.AddBookToList(r.Context(), listID, input.BookID)
	if err != nil {
		writeAddBookToListError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCreateListError(w http.ResponseWriter, err error) {
	if errors.Is(err, lists.ErrNameRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Error(w, "failed to create list", http.StatusInternalServerError)
}

func writeAddBookToListError(w http.ResponseWriter, err error) {
	if errors.Is(err, lists.ErrListNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, lists.ErrBookNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, lists.ErrBookAlreadyInList) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	http.Error(w, "failed to add book to list", http.StatusInternalServerError)
}
