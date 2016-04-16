package controllers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/onesimus-systems/packet-guardian/src/auth"
	"github.com/onesimus-systems/packet-guardian/src/common"
	"github.com/onesimus-systems/packet-guardian/src/models"
)

type Auth struct {
	e *common.Environment
}

func NewAuthController(e *common.Environment) *Auth {
	return &Auth{e: e}
}

func (a *Auth) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/login", a.loginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", a.logoutHandler).Methods("GET")
}

func (a *Auth) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		a.showLoginPage(w, r)
	} else if r.Method == "POST" {
		a.loginUser(w, r)
	}
}

func (a *Auth) showLoginPage(w http.ResponseWriter, r *http.Request) {
	loggedin := a.e.Sessions.GetSession(r).GetBool("loggedin", false)
	if loggedin {
		http.Redirect(w, r, "/manage", http.StatusTemporaryRedirect)
		return
	}
	if err := a.e.Views.NewView("login", r).Render(w, nil); err != nil {
		a.e.Log.Error(err.Error())
	}
}

func (a *Auth) loginUser(w http.ResponseWriter, r *http.Request) {
	// Assume invalid until convinced otherwise
	resp := common.NewAPIResponse(common.APIStatusInvalidAuth, "Invalid login", nil)
	username := r.FormValue("username")
	if auth.IsValidLogin(a.e.DB, username, r.FormValue("password")) {
		resp.Code = common.APIStatusOK
		resp.Message = ""
		sess := common.GetSessionFromContext(r)
		sess.Set("loggedin", true)
		sess.Set("username", username)
		sess.Save(r, w)
		// The context user is not filled in here
		if common.StringInSlice(username, a.e.Config.Auth.AdminUsers) {
			a.e.Log.Infof("Successful login by administrative user %s", username)
		} else {
			a.e.Log.Infof("Successful login by user %s", username)
		}
	} else {
		a.e.Log.Errorf("Failed login attempt by user %s", username)
	}
	resp.WriteTo(w)
}

// LogoutHandler voids a user's session
func (a *Auth) logoutHandler(w http.ResponseWriter, r *http.Request) {
	sess := common.GetSessionFromContext(r)
	if sess.GetBool("loggedin", false) {
		user := models.GetUserFromContext(r)
		sess.Set("loggedin", false)
		sess.Set("username", "")
		sess.Save(r, w)
		a.e.Log.Infof("Successful logout by user %s", user.Username)
	}
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}
