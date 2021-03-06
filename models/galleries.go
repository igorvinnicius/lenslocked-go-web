package models

import (
	"github.com/jinzhu/gorm"
)

type Gallery struct {
	gorm.Model
	UserId uint `gorm:"not null;index"`
	Title string `gorm:"not null"`
}

type GalleryService interface {
	GalleryDB
}

type GalleryDB interface {	
	ByID(id uint) (*Gallery, error)
	ByUserID(userID uint) ([]Gallery, error)
	Create(gallery *Gallery) error
	Update(gallery *Gallery) error
	Delete(id uint) error
}

func (gg *galleryGorm) ByID(id uint) (*Gallery, error) {
	var gallery Gallery
	db := gg.db.Where("id = ?", id)
	err := first(db, &gallery)
	return &gallery, err
}

func (gg *galleryGorm) ByUserID(id uint) ([]Gallery, error) {
	var galleries []Gallery
	gg.db.Where("user_id = ?", id).Find(&galleries)
	return galleries, nil
}

func NewGalleryService(db *gorm.DB) GalleryService {
	return &galleryService {
		GalleryDB: &galleryValidator{
			&galleryGorm{db},
		},
	}
}

type galleryService struct {
	GalleryDB
}

type galleryValidator struct {
	GalleryDB
}

func (gv *galleryValidator) Create(gallery *Gallery) error {	

	err := runGalleryValFuncs(gallery, 
		gv.userIdRequired,
		gv.titleRequired);
	
		if err != nil {
		return err
	}		

	return gv.GalleryDB.Create(gallery)

}

func (gv *galleryValidator) Update(gallery *Gallery) error {	

	err := runGalleryValFuncs(gallery, 
		gv.userIdRequired,
		gv.titleRequired);
	
		if err != nil {
		return err
	}		

	return gv.GalleryDB.Update(gallery)

}

func (gv *galleryValidator) Delete(id uint) error {
	
	var gallery Gallery
	gallery.ID = id

	if id <= 0 {
		return ErrIDInvalid
	}

	return gv.GalleryDB.Delete(id)
}


func (gv *galleryValidator) userIdRequired(gallery *Gallery) error {

	if gallery.UserId <= 0 {
		return ErrUserIdRequired
	}
	return nil
}

func (gv *galleryValidator) titleRequired(gallery *Gallery) error {
	
	if gallery.Title == "" {
		return ErrTitleRequired
	}
	return nil
}

var _ GalleryDB = &galleryGorm{}

type galleryGorm struct {
	db *gorm.DB
}

func (gg *galleryGorm) Create(gallery *Gallery) error {
	return gg.db.Create(gallery).Error
}

func (gg *galleryGorm) Update(gallery *Gallery) error {
	return gg.db.Save(gallery).Error
}

func (gg *galleryGorm) Delete(id uint) error {
	gallery := Gallery{Model: gorm.Model{ID: id}}	
	return gg.db.Delete(&gallery).Error
}

type galleryValFunc func(*Gallery) error

func runGalleryValFuncs(gallery *Gallery, fns ...galleryValFunc) error {

	for _, fn := range fns {
		if err := fn(gallery); err != nil {
			return err
		}
	}

	return nil
}


