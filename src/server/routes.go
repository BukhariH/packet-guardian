package server

import (
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/onesimus-systems/packet-guardian/src/auth"
	"github.com/onesimus-systems/packet-guardian/src/common"
	"github.com/onesimus-systems/packet-guardian/src/controllers"
	"github.com/onesimus-systems/packet-guardian/src/dhcp"
	"github.com/onesimus-systems/packet-guardian/src/models"
	"github.com/onesimus-systems/packet-guardian/src/server/middleware"
)

func LoadRoutes(e *common.Environment) http.Handler {
	r := mux.NewRouter()

	bh := &baseHandlers{e: e}
	r.HandleFunc("/", bh.rootHandler)
	r.NotFoundHandler = http.HandlerFunc(bh.notFoundHandler)
	r.PathPrefix("/public").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("./public/"))))

	controllers.NewDevController(e).RegisterRoutes(r)
	controllers.NewAuthController(e).RegisterRoutes(r)
	controllers.NewManagerController(e).RegisterRoutes(r)
	controllers.NewDeviceController(e).RegisterRoutes(r)
	// controllers.NewAdminController(e).RegisterRoutes(r)

	// We're done with Gorilla's special router, convert to an http.Handler
	h := middleware.SetSessionInfo(e, r)
	h = middleware.Logging(e, h)

	return h
}

type baseHandlers struct {
	e *common.Environment
}

func (b *baseHandlers) rootHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr, ":")[0]
	reg, err := dhcp.IsRegisteredByIP(b.e, net.ParseIP(ip))
	if err != nil {
		b.e.Log.Errorf("Error checking auto registration IP: %s", err.Error())
	}

	if auth.IsLoggedIn(r) {
		if models.GetUserFromContext(r).IsAdmin() {
			http.Redirect(w, r, "/manage", http.StatusTemporaryRedirect)
			//http.Redirect(w, r, "/admin", http.StatusTemporaryRedirect)
		} else {
			http.Redirect(w, r, "/manage", http.StatusTemporaryRedirect)
		}
		return
	}

	if reg {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/register", http.StatusTemporaryRedirect)
	}
}

func (b *baseHandlers) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	if models.GetUserFromContext(r).IsAdmin() {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		//http.Redirect(w, r, "/admin", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	}
}
