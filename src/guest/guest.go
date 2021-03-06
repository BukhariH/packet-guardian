// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package guest

import (
	"bytes"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/lfkeitel/verbose"
	"github.com/usi-lfkeitel/packet-guardian/src/common"
	"github.com/usi-lfkeitel/packet-guardian/src/models"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	guestCodeChars  = "ABCDEFGHJKLMNPQRTUVWXYZ0123456789"
	guestCodeLength = 6
)

// GenerateGuestCode will create a 6 character verification code for guest registrations.
// Possibly confusing letters have been removed. In particular, the letters I, S, and O.
func GenerateGuestCode() string {
	code := bytes.Buffer{}
	for i := 0; i < guestCodeLength; i++ {
		code.WriteByte(guestCodeChars[rand.Intn(len(guestCodeChars))])
	}
	return code.String()
}

// RegisterDevice will register the device for a guest. It is a simplified form of the
// full registration function found in controllers.api.Device.RegistrationHandler().
func RegisterDevice(e *common.Environment, name, credential string, r *http.Request) error {
	// Build guest user model
	guest, err := models.GetUserByUsername(e, credential)
	if err != nil {
		e.Log.WithFields(verbose.Fields{
			"error":    err,
			"package":  "guest",
			"username": credential,
		}).Error("Error getting guest")
		return err
	}
	defer guest.Release()
	guest.DeviceLimit = models.UserDeviceLimit(e.Config.Guest.DeviceLimit)
	guest.DeviceExpiration = &models.UserDeviceExpiration{}

	switch e.Config.Guest.DeviceExpirationType {
	case "never":
		guest.DeviceExpiration.Mode = models.UserDeviceExpirationNever
	case "date":
		guest.DeviceExpiration.Mode = models.UserDeviceExpirationSpecific
		expTime, err := time.ParseInLocation(common.TimeFormat, e.Config.Guest.DeviceExpiration, time.Local)
		if err != nil {
			e.Log.WithFields(verbose.Fields{
				"error":   err,
				"package": "guest",
			}).Error("Error parsing time")
			return errors.New("Internal Server Error")
		}
		guest.DeviceExpiration.Value = expTime.Unix()
	case "duration":
		guest.DeviceExpiration.Mode = models.UserDeviceExpirationDuration
		dur, err := time.ParseDuration(e.Config.Guest.DeviceExpiration)
		if err != nil {
			e.Log.WithFields(verbose.Fields{
				"error":   err,
				"package": "guest",
			}).Error("Error parsing time")
			return errors.New("Internal Server Error")
		}
		guest.DeviceExpiration.Value = int64(dur / time.Second)
	case "daily":
		var err error
		guest.DeviceExpiration.Mode = models.UserDeviceExpirationDaily
		guest.DeviceExpiration.Value, err = common.ParseTime(e.Config.Guest.DeviceExpiration)
		if err != nil {
			e.Log.WithFields(verbose.Fields{
				"error":   err,
				"package": "guest",
			}).Error("Error parsing time")
			return errors.New("Internal Server Error")
		}
	default:
		return errors.New(e.Config.Guest.DeviceExpirationType + " is not a valid device expiration type")
	}

	// Get and enforce the device limit
	deviceCount, err := models.GetDeviceCountForUser(e, guest)
	if err != nil {
		e.Log.WithFields(verbose.Fields{
			"package": "guest",
			"error":   err,
		}).Error("Error getting device count")
	}
	if guest.DeviceLimit != models.UserDeviceLimitUnlimited && deviceCount >= int(guest.DeviceLimit) {
		return errors.New("Device limit reached")
	}

	// Get MAC address
	var mac net.HardwareAddr
	ip := common.GetIPFromContext(r)

	// Automatic registration
	lease, err := models.GetLeaseStore(e).GetLeaseByIP(ip)
	if err != nil {
		e.Log.WithFields(verbose.Fields{
			"error":   err,
			"package": "guest",
			"ip":      ip.String(),
		}).Error("Error getting MAC for IP")
		return errors.New("Internal Server Error")
	} else if lease.ID == 0 {
		e.Log.WithFields(verbose.Fields{
			"package": "guest",
			"ip":      ip.String(),
		}).Notice("Attempted auto reg from non-leased device")
		return errors.New("Error detecting MAC address")
	}
	mac = lease.MAC

	// Get device from database
	device, err := models.GetDeviceByMAC(e, mac)
	if err != nil {
		e.Log.WithFields(verbose.Fields{
			"error":   err,
			"package": "guest",
			"mac":     mac.String(),
		}).Error("Error getting device")
		return errors.New("Database error")
	}

	// Check if device is already registered
	if device.ID != 0 {
		e.Log.WithFields(verbose.Fields{
			"package":  "guest",
			"mac":      mac.String(),
			"username": credential,
		}).Notice("Attempted duplicate registration")
		return errors.New("This device is already registered")
	}

	// Validate platform, we don't want someone to submit an inappropiate value
	platform := common.ParseUserAgent(r.UserAgent())

	// Fill in device information
	device.Username = credential
	device.Description = "Guest - " + name
	device.RegisteredFrom = ip
	device.Platform = platform
	device.Expires = guest.DeviceExpiration.NextExpiration(e, time.Now())
	device.DateRegistered = time.Now()
	device.LastSeen = time.Now()
	device.UserAgent = r.UserAgent()

	// Save new device
	if err := device.Save(); err != nil {
		e.Log.WithFields(verbose.Fields{
			"error":   err,
			"package": "guest",
		}).Error("Error saving device")
		return errors.New("Error registering device")
	}
	e.Log.WithFields(verbose.Fields{
		"package":  "guest",
		"mac":      mac.String(),
		"name":     name,
		"username": credential,
		"action":   "register-guest-device",
	}).Info("Device registered")
	return nil
}
