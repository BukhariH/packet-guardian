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
	"strings"
	"time"

	"github.com/onesimus-systems/packet-guardian/src/common"
	"github.com/onesimus-systems/packet-guardian/src/dhcp"
	"github.com/onesimus-systems/packet-guardian/src/models"
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
		e.Log.WithField("Err", err).Error("Failed to get guest user")
		return err
	}
	guest.DeviceLimit = models.UserDeviceLimit(e.Config.Guest.DeviceLimit)
	// TODO: Honor configuration settings
	guest.DeviceExpiration = &models.UserDeviceExpiration{
		Mode:  models.UserDeviceExpirationDaily,
		Value: int64((time.Duration(24) * time.Hour) / time.Second),
	}

	// Get and enforce the device limit
	deviceCount, err := models.GetDeviceCountForUser(e, guest)
	if err != nil {
		e.Log.Errorf("Error getting device count: %s", err.Error())
	}
	if guest.DeviceLimit != models.UserDeviceLimitUnlimited && deviceCount >= int(guest.DeviceLimit) {
		return errors.New("Device limit reached")
	}

	// Get MAC address
	var mac net.HardwareAddr
	ip := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])

	// Automatic registration
	lease, err := dhcp.GetLeaseByIP(e, ip)
	if err != nil {
		e.Log.Errorf("Failed to get MAC for IP %s: %s", ip, err.Error())
		return errors.New("Internal Server Error")
	} else if lease.ID == 0 {
		e.Log.Errorf("Attempted automatic registration on non-leased device %s", ip)
		return errors.New("Error detecting MAC address")
	}
	mac = lease.MAC

	// Get device from database
	device, err := models.GetDeviceByMAC(e, mac)
	if err != nil {
		e.Log.Errorf("Error getting device: %s", err.Error())
	}

	// Check if device is already registered
	if device.ID != 0 {
		e.Log.Noticef("Attempted duplicate registration of MAC %s to user %s", mac.String(), credential)
		return errors.New("This device is already registered")
	}

	// Validate platform, we don't want someone to submit an inappropiate value
	platform := common.ParseUserAgent(r.UserAgent())

	// Fill in device information
	device.Username = credential
	device.Description = "Guest - " + name
	device.RegisteredFrom = ip
	device.Platform = platform
	device.Expires = guest.DeviceExpiration.NextExpiration(e)
	device.DateRegistered = time.Now()
	device.LastSeen = time.Now()
	device.UserAgent = r.UserAgent()

	// Save new device
	if err := device.Save(); err != nil {
		e.Log.Errorf("Error registering device: %s", err.Error())
		return errors.New("Error registering device")
	}
	e.Log.Infof("Successfully registered MAC %s to guest %s <%s>", mac.String(), name, credential)
	return nil
}
