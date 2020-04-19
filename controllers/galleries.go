package controllers

import (	
	"fmt"
	"log"
	"strconv"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/igorvinnicius/lenslocked-go-web/views"
	"github.com/igorvinnicius/lenslocked-go-web/models"
	"github.com/igorvinnicius/lenslocked-go-web/context"
)

const(
	ShowGallery = "show_gallery"
)

func NewGalleries(galleryService models.GalleryService, r *mux.Router) *Galleries {
	return &Galleries{
		New: views.NewView("bootstrap", "galleries/new"),
		ShowView: views.NewView("bootstrap", "galleries/show"),			
		GalleryService : galleryService,
		r: r,
	}
}

type Galleries struct{
	New *views.View	
	ShowView *views.View
	GalleryService models.GalleryService
	r *mux.Router
}

type GalleryForm struct {
	Title string `schema:"title"`	
}

func (g *Galleries) Show(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid gallery id", http.StatusNotFound)
	}

	gallery, err := g.GalleryService.ByID(uint(id))
	if err != nil {
		switch err {
		case models.ErrNotFound:
			http.Error(w, "Gallery not found", http.StatusNotFound)
			return
		default:
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	var vd views.Data
	vd.Yield = gallery
	g.ShowView.Render(w, vd)	
}

func (g *Galleries) Create(w http.ResponseWriter, r *http.Request) {
		
	var vd views.Data
	var form GalleryForm

	if err := parseForm(r, &form); err != nil {
		log.Println(err)
		vd.SetAlert(err)
		g.New.Render(w, vd)
		return
	}

	user := context.User(r.Context())

	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	gallery := models.Gallery {
		Title: form.Title,
		UserId: user.ID,
	}
	
	if err := g.GalleryService.Create(&gallery); err != nil {		
		vd.SetAlert(err)
		g.New.Render(w, vd)
		return		
	}

	url, err := g.r.Get(ShowGallery).URL("id", fmt.Sprintf("%v", gallery.ID))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}