package controllers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/igorvinnicius/lenslocked-go-web/context"
	"github.com/igorvinnicius/lenslocked-go-web/models"
	"github.com/igorvinnicius/lenslocked-go-web/views"
)

const (
	ShowGallery = "show_gallery"
	EditGallery = "edit_gallery"

	maxMultipartMem = 1 << 20
)

func NewGalleries(galleryService models.GalleryService, r *mux.Router) *Galleries {
	return &Galleries{
		New:            views.NewView("bootstrap", "galleries/new"),
		ShowView:       views.NewView("bootstrap", "galleries/show"),
		EditView:       views.NewView("bootstrap", "galleries/edit"),
		IndexView:      views.NewView("bootstrap", "galleries/index"),
		GalleryService: galleryService,
		r:              r,
	}
}

type Galleries struct {
	New            *views.View
	ShowView       *views.View
	EditView       *views.View
	IndexView      *views.View
	GalleryService models.GalleryService
	r              *mux.Router
}

type GalleryForm struct {
	Title string `schema:"title"`
}

func (g *Galleries) Index(w http.ResponseWriter, r *http.Request) {

	user := context.User(r.Context())
	galleries, err := g.GalleryService.ByUserID(user.ID)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var vd views.Data
	vd.Yield = galleries
	g.IndexView.Render(w, r, vd)
}

func (g *Galleries) Show(w http.ResponseWriter, r *http.Request) {

	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}

	var vd views.Data
	vd.Yield = gallery
	g.ShowView.Render(w, r, vd)
}

func (g *Galleries) Edit(w http.ResponseWriter, r *http.Request) {

	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}

	user := context.User(r.Context())
	if gallery.UserId != user.ID {
		http.Error(w, "GAllery not found", http.StatusNotFound)
		return
	}

	var vd views.Data
	vd.Yield = gallery
	vd.User = user
	g.EditView.Render(w, r, vd)
}

func (g *Galleries) Update(w http.ResponseWriter, r *http.Request) {

	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}

	user := context.User(r.Context())
	if gallery.UserId != user.ID {
		http.Error(w, "GAllery not found", http.StatusNotFound)
		return
	}

	var vd views.Data
	vd.Yield = gallery
	var form GalleryForm

	if err := parseForm(r, &form); err != nil {
		log.Println(err)
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}

	gallery.Title = form.Title

	err = g.GalleryService.Update(gallery)
	if err != nil {
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}

	vd.Alert = &views.Alert{
		Level:   views.AlertLevelSuccess,
		Message: "Gallery successfully updated",
	}

	g.EditView.Render(w, r, vd)
}

func (g *Galleries) Delete(w http.ResponseWriter, r *http.Request) {

	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}

	user := context.User(r.Context())
	if gallery.UserId != user.ID {
		http.Error(w, "GAllery not found", http.StatusNotFound)
		return
	}

	var vd views.Data

	err = g.GalleryService.Delete(gallery.ID)
	if err != nil {
		vd.SetAlert(err)
		vd.Yield = gallery
		g.EditView.Render(w, r, vd)
		return
	}

	http.Redirect(w, r, "/galleries", http.StatusFound)
}

func (g *Galleries) Create(w http.ResponseWriter, r *http.Request) {

	var vd views.Data
	var form GalleryForm

	if err := parseForm(r, &form); err != nil {
		log.Println(err)
		vd.SetAlert(err)
		g.New.Render(w, r, vd)
		return
	}

	user := context.User(r.Context())

	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	gallery := models.Gallery{
		Title:  form.Title,
		UserId: user.ID,
	}

	if err := g.GalleryService.Create(&gallery); err != nil {
		vd.SetAlert(err)
		g.New.Render(w, r, vd)
		return
	}

	url, err := g.r.Get(EditGallery).URL("id", fmt.Sprintf("%v", gallery.ID))
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, url.Path, http.StatusFound)
}

func (g *Galleries) ImageUpload(w http.ResponseWriter, r *http.Request) {

	gallery, err := g.galleryByID(w, r)
	if err != nil {
		return
	}

	user := context.User(r.Context())
	if gallery.UserId != user.ID {
		http.Error(w, "GAllery not found", http.StatusNotFound)
		return
	}

	var vd views.Data
	vd.Yield = gallery
	// var form GalleryForm

	//TODO: parse the mutipart form

	err = r.ParseMultipartForm(maxMultipartMem)
	if err != nil {
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}

	// create a directory to upload file
	galleryPath := fmt.Sprintf("images/galleries/%v/", gallery.ID)
	err = os.MkdirAll(galleryPath, 0755)
	if err != nil {
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}

	files := r.MultipartForm.File["images"]
	for _, f := range files {
		//open the uploaded file
		file, err := f.Open()
		if err != nil {
			vd.SetAlert(err)
			g.EditView.Render(w, r, vd)
			return
		}
		defer file.Close()

		// create a destination file
		dest, err := os.Create(galleryPath + f.Filename)
		if err != nil {
			vd.SetAlert(err)
			g.EditView.Render(w, r, vd)
			return
		}
		dest.Close()

		// copy uploaded file data to the destination file
		_, err = io.Copy(dest, file)
		if err != nil {
			vd.SetAlert(err)
			g.EditView.Render(w, r, vd)
			return
		}
		fmt.Fprintln(w, "Files successfully uploaded!")
	}
}

func (g *Galleries) galleryByID(w http.ResponseWriter, r *http.Request) (*models.Gallery, error) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid gallery id", http.StatusNotFound)
		return nil, err
	}

	gallery, err := g.GalleryService.ByID(uint(id))
	if err != nil {
		switch err {
		case models.ErrNotFound:
			http.Error(w, "Gallery not found", http.StatusNotFound)
		default:
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
		return nil, err
	}

	return gallery, nil
}
