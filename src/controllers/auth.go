package controllers

import (
	"net/http"

	"github.com/onesimus-systems/packet-guardian/src/auth"
	"github.com/onesimus-systems/packet-guardian/src/common"
)

type Auth struct {
	e *common.Environment
}

func NewAuthController(e *common.Environment) *Auth {
	return &Auth{e: e}
}

func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
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
	a.e.Views.NewView("login", r).Render(w, nil)
}

func (a *Auth) loginUser(w http.ResponseWriter, r *http.Request) {
	// Assume invalid until convinced otherwise
	resp := common.NewAPIResponse(common.APIStatusInvalidAuth, "Invalid login", nil)
	if auth.LoginUser(r, w) {
		resp.Code = common.APIStatusOK
		resp.Message = ""
	}
	resp.WriteTo(w)
}

// LogoutHandler voids a user's session
func (a *Auth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	auth.LogoutUser(r, w)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}
