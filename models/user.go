package models

import(
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"golang.org/x/crypto/bcrypt"

	"github.com/igorvinnicius/lenslocked-go-web/hash"
	"github.com/igorvinnicius/lenslocked-go-web/rand"
)

var(
	ErrNotFound = errors.New("models: resource not found")
	ErrInvalidID = errors.New("models: ID must me > 0")	
	ErrInvalidPassword = errors.New("models: incorrect password provided")
)

const userPwPepper = "secret-random-string"
const hmacSecretKey = "secret-hmac-key"

type UserDB interface {
	ById(id uint) (*User, error)
	ByEmail(email string) (*User, error)
	ByRemember(token string) (*User, error)

	Create(user *User) error
	Update(user *User) error
	Delete(id uint) error

	Close() error

	AutoMigrate() error
	DestructiveReset() error
}

type UserService interface {

	Authenticate(email, password string) (*User, error)
	UserDB
}

type User struct {
	gorm.Model
	Name string
	Email string `gorm:"not null;unique_index"`
	Password string `gorm:"-"`
	PasswordHash string `gorm:"not null"`
	Remember string `gorm:"-"`
	RememberHash string `gorm:"not null;unique_index"`
}

func NewUserService(connectionInfo string) (UserService, error) {
	
	ug, err := newUserGorm(connectionInfo)
	if err != nil {
		return nil, err
	}

	hmac := hash.NewHMAC(hmacSecretKey)

	uv := &userValidator{
		hmac: hmac,
		UserDB: ug,
	}

	return &userService {
		UserDB : uv,
	}, nil
}

func newUserGorm(connectionInfo string) (*userGorm, error) {
	db, err := gorm.Open("postgres", connectionInfo)
	if err != nil {
		return nil, err
	}

	db.LogMode(true)	

	return &userGorm{
		db: db,	
	}, nil
}

var _ UserService = &userService{}

type userService struct {
	UserDB
}

type userValFunc func(*User) error

func runUserValFuncs(user *User, fns ...userValFunc) error {

	for _, fn := range fns{
		if err := fn(user); err != nil {
			return err
		}
	}

	return nil
}

var _ UserDB = &userValidator{}

type userValidator struct {
	UserDB
	hmac hash.HMAC
}

func (uv *userValidator) Create(user *User) error {

	if user.Remember == "" {
		
		token, err := rand.RememberToken()
		if err != nil {
			return err
		}

		user.Remember = token		
	}

	err := runUserValFuncs(user, uv.bcryptPassword, uv.hmacRemember);
	if err != nil {
		return err
	}		

	return uv.UserDB.Create(user)
}

func (uv *userValidator) Update(user *User) error {

	err := runUserValFuncs(user, uv.bcryptPassword, uv.hmacRemember);
	if err != nil {
		return err
	}	

	return uv.UserDB.Update(user)
}

func (uv *userValidator) Delete(id uint) error {
	
	if id == 0 {
		return ErrInvalidID
	}

	return uv.UserDB.Delete(id)
}

func (uv *userValidator) bcryptPassword(user *User) error {

	if user.Password == "" {
		return nil
	}

	passwordBytes := []byte(user.Password + userPwPepper)

	hashedBytes, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedBytes)
	user.Password = ""

	return nil
}

func (uv *userValidator) hmacRemember(user *User) error {

	if user.Remember == "" {
		return nil
	}

	user.RememberHash = uv.hmac.Hash(user.Remember)

	return nil
}

func (uv *userValidator) ByRemember(token string) (*User, error) {
	
	user := User{
		Remember: token,
	}

	if err := runUserValFuncs(&user, uv.hmacRemember); err != nil {
		return nil, err
	}

	return uv.UserDB.ByRemember(user.RememberHash)
}

var _ UserDB = &userGorm{}

type userGorm struct {
	db *gorm.DB	
}

func (ug *userGorm) ById(id uint) (*User, error) {
	var user User
	db := ug.db.Where("id = ?", id)
	err := first(db, &user)
	return &user, err
}

func (ug *userGorm) ByEmail(email string) (*User, error) {
	var user User
	db := ug.db.Where("email = ?", email)
	err := first(db, &user)
	return &user, err
}

func (ug *userGorm) ByRemember(rememberHash string) (*User, error) {
	var user User	
	db := ug.db.Where("remember_hash = ?", rememberHash)
	err := first(db, &user)
	return &user, err
}

func first(db *gorm.DB, dest interface{}) error {
	
	err := db.First(dest).Error
	
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}

	return err
}

func (us *userService) Authenticate(email, password string) (*User, error) {
	
	foundUser, err := us.ByEmail(email)

	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(password + userPwPepper))

	if err != nil {
		switch err {
			case bcrypt.ErrMismatchedHashAndPassword:
				return nil, ErrInvalidPassword	
			default:
				return nil, err
		}
	}

	return foundUser, nil
}

func (ug *userGorm) Create(user *User) error {

	return ug.db.Create(user).Error

}

func (ug *userGorm) Update(user *User) error {

	return ug.db.Save(user).Error
}

func (ug *userGorm) Delete(id uint) error {
	
	if id == 0 {
		return ErrInvalidID
	}

	user := User{Model: gorm.Model{ID: id}}
	
	return ug.db.Delete(&user).Error
}


func (ug *userGorm) Close() error {
	return ug.db.Close()
}

func (ug *userGorm) DestructiveReset() error {
	
	if err := ug.db.DropTableIfExists(&User{}).Error; err != nil {
		return err
	}

	return ug.AutoMigrate()
}

func (ug *userGorm) AutoMigrate() error {
	
	if err := ug.db.AutoMigrate(&User{}).Error; err != nil {
		return err
	}

	return nil
}