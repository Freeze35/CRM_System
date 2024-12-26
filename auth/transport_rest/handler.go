package transport_rest

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type AuthService interface {
	Login(ctx context.Context) error
	Auth(ctx context.Context) error
}

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) InitRouter() *mux.Router {
	r := mux.NewRouter()

	books := r.PathPrefix("/auth").Subrouter()
	{
		books.HandleFunc("/login", h.Login).Methods(http.MethodPost)
		books.HandleFunc("/authin", h.AuthIn).Methods(http.MethodGet)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	return r
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	//id, err := getIdFromRequest(r)
	/*if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	book, err := h.booksService.GetByID(context.TODO(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBookNotFound) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(book)
	if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")*/
	response, _ := json.Marshal("dd")
	w.Write(response)
}

func (h *Handler) AuthIn(w http.ResponseWriter, r *http.Request) {
	log.Printf("Restro")
	response, _ := json.Marshal("dd")
	w.Write(response)
	/*id, err := getIdFromRequest(r)
	if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	book, err := h.booksService.GetByID(context.TODO(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBookNotFound) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(book)
	if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(response)*/
}
