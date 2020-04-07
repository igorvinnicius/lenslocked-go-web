package controllers

import (
	"fmt"
	"net/http"
	"github.com/igorvinnicius/lenslocked-go-web/views"
	"github.com/igorvinnicius/lenslocked-go-web/models"
)

type SignupForm struct {
	Name string `schema:"name"`
	Email string `schema:"email"`
	Password string `schema:"password"`
}

func NewUsers(userService *models.UserService) *Users {
	return &Users{
		NewView: views.NewView("bootstrap", "users/new"),
		UserService : userService,	
	}
}

type Users struct{
	NewView *views.View
	UserService *models.UserService
}

func (u *Users) New(w http.ResponseWriter, r *http.Request){
	if err := u.NewView.Render(w, nil); err != nil {
		panic(err)
	}
}

func (u *Users) Create(w http.ResponseWriter, r *http.Request){
		
	var form SignupForm

	if err := parseForm(r, &form); err != nil {
		panic(err)
	}

	user := models.User {
		Name: form.Name,
		Email: form.Email,
		Password: form.Password,
	}
	
	if err:= u.UserService.Create(&user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}

	fmt.Fprintln(w, user)	
}