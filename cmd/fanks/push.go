
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/labstack/echo/v4"
	"github.com/oliverisaac/fanks/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const webPushVapidPublicKey = "BL4I9zM4s2B4v_2kpt2bTCNuWJkXzT5LPZ2sA2a-2p2l5g3aH-t8B8g8G2f0f2a8B6E8F0G2A4C6E8G0I2"
const webPushVapidPrivateKey = "BL4I9zM4s2B4v_2kpt2bTCNuWJkXzT5LPZ2sA2a-2p2l5g3aH-t8B8g8G2f0f2a8B6E8F0G2A4C6E8G0I2"

func sendPushNotifications(db *gorm.DB) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			now := time.Now()
			loc, err := time.LoadLocation("America/Chicago")
			if err != nil {
				logrus.Error(errors.Wrap(err, "loading location"))
				continue
			}
			if now.In(loc).Hour() == 21 && now.In(loc).Minute() == 00 {
				users, err := getAllUsersWithSubscriptions(db)
				if err != nil {
					logrus.Error(errors.Wrap(err, "getting all users"))
					continue
				}

				for _, user := range users {
					sendPushNotificationToUser(user)
				}
			}
		}
	}()
}

func getAllUsersWithSubscriptions(db *gorm.DB) ([]types.User, error) {
	var users []types.User
	err := db.Preload("PushSubscriptions").Find(&users).Error
	return users, err
}

func sendPushNotificationToUser(user types.User) {
	for _, subData := range user.PushSubscriptions {
		sub := &webpush.Subscription{
			Endpoint: subData.Endpoint,
			Keys: webpush.Keys{
				P256dh: subData.P256DH,
				Auth:   subData.Auth,
			},
		}

		resp, err := webpush.SendNotification([]byte("{\"title\":\"Fanks\",\"body\":\"What are you thankful for today?\"}"), sub, &webpush.Options{
			VAPIDPublicKey:  webPushVapidPublicKey,
			VAPIDPrivateKey: webPushVapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			logrus.Error(errors.Wrap(err, "sending push notification"))
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("Sent push notification to user %s\n", user.Email)
	}
}

func saveSubscription(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if !ok {
			return c.String(http.StatusUnauthorized, "unauthorized")
		}

		var sub webpush.Subscription
		if err := c.Bind(&sub); err != nil {
			return errors.Wrap(err, "binding subscription")
		}

		keys, err := json.Marshal(sub.Keys)
		if err != nil {
			return errors.Wrap(err, "marshalling subscription keys")
		}

		pushSubscription := types.PushSubscription{
			UserID:   user.ID,
			Endpoint: sub.Endpoint,
			P256DH:   sub.Keys.P256dh,
			Auth:     sub.Keys.Auth,
			Keys:     string(keys),
		}

		if err := db.Create(&pushSubscription).Error; err != nil {
			return errors.Wrap(err, "saving subscription")
		}

		return c.String(http.StatusOK, "subscription saved")
	}
}
