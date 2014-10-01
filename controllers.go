package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"log"

	"github.com/stretchr/goweb"
	"github.com/stretchr/goweb/context"
)

type Getter interface {
	Get(id string) (error, interface{})
}

// Controllers for GUILD eyewear lego site
// Uses goweb github.com/stretchr/goweb
type (
	accountController   struct{}
	userController      struct{}
	materialsController struct{}
)

// Authorization
func requireAuth(requiredUserLevel byte, ctx context.Context) bool {
	log.Println("Checking if user has authorization")
	user := ctx.Data()["user"]
	log.Printf("User is %v", user)
	if user == nil {
		log.Println("Returning unauthorized")
		return false
	}
	ut := user.(User).Type
	if ut|requiredUserLevel == 0 {
		return false
	}
	return true
}

// Materials controller

func (m *materialsController) Read(id string, ctx context.Context) error {
	mat, err := findMaterialById(id)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 200, mat)
}
func (m *materialsController) Create(ctx context.Context) error {
	if !requireAuth(USER_SYSTEM_ADMIN, ctx) {
		return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
	}

	var mat Material
	data, err := ctx.RequestBody()

	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := json.Unmarshal(data, &mat); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := createMaterial(&mat); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 301, mat)
}

func (m *materialsController) ReadMany(ctx context.Context) error {
	log.Println("Getting all materials")
	materials, err := getAllMaterials()
	log.Printf("Materials %v", materials)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.RespondWithData(ctx, materials)
}

// Account controller functions
func (a *accountController) Read(id string, ctx context.Context) error {
	log.Println("Getting user")
	// Get the authenticated user
	user := ctx.Data()["user"]
	return goweb.API.WriteResponseObject(ctx, 200, user)
}

func (a *accountController) Create(ctx context.Context) error {
	var acct Account
	data, err := ctx.RequestBody()
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := json.Unmarshal(data, &acct); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := createAccount(&acct); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return a.Read(acct.Id.Hex(), ctx)
}

// User controller functions
func (u *userController) Read(id string, ctx context.Context) error {
	log.Println("Getting user")
	// Get the authenticated user
	userdata := ctx.Data()["user"]
	if userdata == nil {
		return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
	}
	loggedin_user := userdata.(User)

	var requested_user User
	var err error

	switch loggedin_user.Type {
	case USER_NORMAL:
		if loggedin_user.Id == id {
			requested_user = loggedin_user
		} else {
			return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
		}
	case USER_SYSTEM_ADMIN:
		if requested_user, err = findUserById(id); err != nil {
			return goweb.API.RespondWithError(ctx, 500, err.Error())
		}
	case USER_ACCOUNT_ADMIN:
		if requested_user, err = findUserById(id); err != nil {
			return goweb.API.RespondWithError(ctx, 500, err.Error())
		}
		if requested_user.AccountId != loggedin_user.AccountId {
			return goweb.API.RespondWithError(ctx, 404, "Not Found")
		}
	}

	return goweb.API.RespondWithData(ctx, requested_user)
}

func (u *userController) Create(ctx context.Context) error {
	var user User
	userInfo := ctx.FormParams()
	if len(userInfo) > 0 {
		if userInfo["id"] != nil {
			user.Id = userInfo["email"].(string)
			log.Printf("Obtained id %v", user.Id)
		}
		if userInfo["password"] != nil {
			user.Password = userInfo["password"].(string)
			log.Printf("Obtained pw %v", user.Password)
		}
	} else { // Read JSON data
		data, err := ctx.RequestBody()
		if err != nil {
			return goweb.API.RespondWithError(ctx, 400, err.Error())
		}

		if err := json.Unmarshal(data, &user); err != nil {
			return goweb.API.RespondWithError(ctx, 400, err.Error())
		}
		log.Printf("Unmarshalled user %v", user)
	}
	if len(user.Id) == 0 || len(user.Password) == 0 {
		return goweb.API.RespondWithError(ctx, 400, "email and password required")
	}
	_, err := findUserById(user.Id)
	if err == nil {
		return goweb.API.RespondWithError(ctx, 409, "user already exists")
	}

	// Generate salt
	salt := make([]byte, 16)
	var n int
	if n, err = rand.Read(salt); err != nil {
		return err
	}
	user.PwSalt = string(salt[:n])
	saltedpassword := (user.PwSalt + user.Password)
	hash := sha512.New()
	hash.Write([]byte(saltedpassword))
	user.PwHash = hex.EncodeToString(hash.Sum(nil))

	if err := createUser(&user); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	return u.Read(user.Id, ctx)
}
