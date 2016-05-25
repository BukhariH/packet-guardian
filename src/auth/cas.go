// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/dragonlibs/cas"
	"github.com/lfkeitel/verbose"

	"github.com/onesimus-systems/packet-guardian/src/common"
	"github.com/onesimus-systems/packet-guardian/src/models"
)

func init() {
	authFunctions["cas"] = &casAuthenticator{}
}

type casAuthenticator struct {
	client *cas.Client
}

func (c *casAuthenticator) loginUser(r *http.Request, w http.ResponseWriter) bool {
	e := common.GetEnvironmentFromContext(r)
	if c.client == nil {
		casUrlStr := strings.TrimRight(e.Config.Auth.CAS.Server, "/") + "/" // Ensure server ends in /
		casUrl, err := url.Parse(casUrlStr)
		if err != nil {
			e.Log.WithFields(verbose.Fields{
				"Err Msg": err,
				"CasURL":  casUrlStr,
			}).Error("Failed to parse CAS url")
			return false
		}
		c.client = &cas.Client{
			URL: casUrl,
		}
	}

	_, err := c.client.AuthenticateUser(r.FormValue("username"), r.FormValue("password"), r)
	if err == cas.InvalidCredentials {
		return false
	}
	if err != nil {
		e.Log.WithField("Err Msg", err).Error("Error communicating with CAS server")
		return false
	}

	user, err := models.GetUserByUsername(e, r.FormValue("username"))
	if err != nil {
		e.Log.WithField("Err", err).Error("Error getting user")
		return false
	}
	if user.IsExpired() {
		e.Log.WithField("username", user.Username).Info("User expired")
		return false
	}

	return true
}
