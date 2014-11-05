package main

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/guildeyewear/legoserver/models"
	"github.com/stretchr/goweb"
	"github.com/stretchr/goweb/context"
	"gopkg.in/mgo.v2"
)

// mapRoutes uses the goweb package to map all our RESTful
// endpoints to function handlers.  It's put into a
// distinct function so that it can be called from test code.
func mapRoutes() {
	//var securePaths = map[string]byte{
	//		"GET: /accounts/*/users": models.USER_NORMAL,
	//	}

	goweb.MapBefore(func(c context.Context) error {
		r := c.HttpRequest()
		log.Printf("%v: %v", r.Method, r.URL.Path)
		authheader := c.HttpRequest().Header["Authorization"]
		if len(authheader) == 0 {
			return nil
		}
		auth := strings.SplitN(authheader[0], " ", 2)
		if len(auth) == 2 && auth[0] == "Basic" {
			authstr, _ := base64.StdEncoding.DecodeString(auth[1])

			creds := strings.SplitN(string(authstr), ":", 2)
			if len(creds) == 2 && len(creds[0]) > 0 && len(creds[1]) > 0 {
				user, err := models.FindUserById(creds[0])
				if err != nil {
					return goweb.API.RespondWithError(c, 500, err.Error())
				} else if user.ValidatePassword(creds[1]) {
					c.Data()["user"] = user
				}
			}
			return nil
		}

		// Do authentication
		return nil
	})
	goweb.MapAfter(func(c context.Context) error {
		// add logging
		return nil
	})
	goweb.Map("/", func(c context.Context) error {
		// home page
		return nil
	})

	// Map controllers
	goweb.MapController("/accounts", &accountController{})
	goweb.MapController("/users", &userController{})
	goweb.MapController("/collections", &collectionsController{})
	goweb.MapController("/materials", &materialsController{})
	goweb.MapController("/orders", &ordersController{})
	goweb.MapController("/designs", &designController{})

	goweb.Map("/accounts/{id}/users", accountUsers)
	goweb.Map("/importdesign", importDesign)
	goweb.Map("/designs/{id}/render", getDesignRender)

	// Map status code responses for testing
	goweb.Map("/status-code/{code}", func(c context.Context) error {
		// get the path value as an integer
		statusCodeInt, statusCodeIntErr := strconv.Atoi(c.PathValue("code"))
		if statusCodeIntErr != nil {
			return goweb.Respond.With(c, http.StatusInternalServerError, []byte("Failed to convert 'code' into a real status code number."))
		}
		// respond with the status
		return goweb.Respond.WithStatusText(c, statusCodeInt)
	})

	// errortest should throw a system error and be handled by the
	// DefaultHttpHandler().ErrorHandler() Handler.
	goweb.Map("/errortest", func(c context.Context) error {
		return errors.New("This is a test error!")
	})

	//	Map the static-files directory so it's exposed as /static
	goweb.MapStatic("/static", "static-files")

	//	Map the a favicon
	goweb.MapStaticFile("/favicon.ico", "static-files/favicon.ico")

	//	Catch-all handler for everything that we don't understand
	goweb.Map(func(c context.Context) error {

		// just return a 404 message
		return goweb.API.Respond(c, 404, nil, []string{"File not found"})

	})

}

func logHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v: %v", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func addRoute(route string, next http.HandlerFunc) {
	http.Handle(route, logHandler(next))
}

func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	// Set up the API responder
	mapRoutes()

	log.Println("Listening Carefully on port", port)
	http.ListenAndServe(":"+port, goweb.DefaultHttpHandler())
}
