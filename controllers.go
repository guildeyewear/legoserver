package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"

	"github.com/guildeyewear/legoserver/models"
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
	ordersController    struct{}
)

// Authorization
func requireAuth(userLevel byte, ctx context.Context) bool {
	log.Println("Checking if user has authorization")
	user := ctx.Data()["user"]
	log.Printf("geometry.User is %v", user)
	if user == nil {
		log.Println("Returning unauthorized")
		return false
	}
	ut := user.(models.User).Type
	if ut == 0 {
		return false
	}
	return true
}

// Orders
func (o *ordersController) Create(ctx context.Context) error {
	if !requireAuth(models.USER_NORMAL, ctx) {
		return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
	}
	var order models.Order
	if data, err := ctx.RequestBody(); err != nil {
		log.Printf("Error getting request body in POST /orders: %v", err)
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	} else {
		if err = json.Unmarshal(data, &order); err != nil {
			log.Printf("Error unmarshalling request body in POST /orders: %v", err)
			return goweb.API.RespondWithError(ctx, 400, err.Error())
		}
		user := ctx.Data()["user"].(models.User)
		order.UserId = user.Id
		order.AccountId = user.AccountId
		if err = models.CreateOrder(&order); err != nil {
			log.Printf("Error creating order in database in POST /orders: %v", err)
			return goweb.API.RespondWithError(ctx, 400, err.Error())
		}
	}
	return goweb.API.WriteResponseObject(ctx, 201, order)
}

func (o *ordersController) Read(id string, ctx context.Context) error {
	order, err := models.FindOrderById(id)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 200, order)
}

func (o *ordersController) ReadMany(ctx context.Context) error {
	var (
		status int64
		err    error
		orders []models.Order
	)
	if orderStatusStr := ctx.FormValue("status"); len(orderStatusStr) > 0 {
		if status, err = strconv.ParseInt(orderStatusStr, 0, 0); err != nil {
			return err
		}
		if orders, err = models.GetOrders(int(status)); err != nil {
			return err
		}
	} else if orders, err = models.GetAllOrders(); err != nil {
		return err
	}
	return goweb.API.WriteResponseObject(ctx, 200, orders)
}

// Materials controller

func (m *materialsController) Read(id string, ctx context.Context) error {
	mat, err := models.FindMaterialById(id)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	log.Printf("Got material %v", mat)
	return goweb.API.WriteResponseObject(ctx, 200, mat)
}
func (m *materialsController) Create(ctx context.Context) error {
	if !requireAuth(models.USER_SYSTEM_ADMIN, ctx) {
		return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
	}

	var mat models.Material
	data, err := ctx.RequestBody()

	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := json.Unmarshal(data, &mat); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := models.CreateMaterial(&mat); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 201, mat)
}

func (m *materialsController) ReadMany(ctx context.Context) error {
	log.Println("Getting all materials")
	materials, err := models.GetAllMaterials()
	log.Printf("Materials %v", materials)
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 200, materials)
}

// Account controller functions
func (a *accountController) Read(id string, ctx context.Context) error {
	log.Println("Getting user")
	// Get the authenticated user
	user := ctx.Data()["user"]
	return goweb.API.WriteResponseObject(ctx, 200, user)
}

func (a *accountController) ReadMany(ctx context.Context) error {
	accounts, err := models.GetAllAccounts()
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 200, accounts)
}

func (a *accountController) Create(ctx context.Context) error {
	var acct models.Account
	data, err := ctx.RequestBody()
	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := json.Unmarshal(data, &acct); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	if err := models.CreateAccount(&acct); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 201, acct)
}

func accountUsers(ctx context.Context) error {
	id := ctx.PathParams().Get("id")
	log.Printf("Getting users for account %v", id)
	users, err := models.FindUsersbyAccount(id.Str())

	if err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}
	return goweb.API.WriteResponseObject(ctx, 200, users)
}

func (u *userController) Read(id string, ctx context.Context) error {
	log.Println("Getting user")
	// Get the authenticated user
	userdata := ctx.Data()["user"]
	if userdata == nil {
		return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
	}
	loggedin_user := userdata.(models.User)

	var requested_user models.User
	var err error

	switch loggedin_user.Type {
	case models.USER_NORMAL:
		if loggedin_user.Id == id {
			requested_user = loggedin_user
		} else {
			return goweb.API.RespondWithError(ctx, 401, "Unauthorized")
		}
	case models.USER_SYSTEM_ADMIN:
		if requested_user, err = models.FindUserById(id); err != nil {
			return goweb.API.RespondWithError(ctx, 500, err.Error())
		}
	case models.USER_ACCOUNT_ADMIN:
		if requested_user, err = models.FindUserById(id); err != nil {
			return goweb.API.RespondWithError(ctx, 500, err.Error())
		}
		if requested_user.AccountId != loggedin_user.AccountId {
			return goweb.API.RespondWithError(ctx, 404, "Not Found")
		}
	}

	return goweb.API.WriteResponseObject(ctx, 200, requested_user)
}

func (u *userController) Create(ctx context.Context) error {
	var user models.User
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
	_, err := models.FindUserById(user.Id)
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

	if err := models.CreateUser(&user); err != nil {
		return goweb.API.RespondWithError(ctx, 400, err.Error())
	}

	return u.Read(user.Id, ctx)
}
