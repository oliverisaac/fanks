package main

import (
	"encoding/json"
	errs "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/labstack/echo/v4"
	"github.com/oliverisaac/fanks/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var triggerPushChan = make(chan uint)

func startNotificationWorker(cfg types.Config, db *gorm.DB) error {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return (errors.Wrap(err, "loading location"))
	}
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			now := time.Now()
			if now.In(loc).Hour() == 21 && now.In(loc).Minute() == 00 {
				triggerPushChan <- 0
			}
		}
	}()

	go func() {
		for triggerUID := range triggerPushChan {
			logrus.Info("Trigging push notifications for all users")
			users, err := getAllUsersWithSubscriptions(db)
			if err != nil {
				logrus.Error(errors.Wrap(err, "getting all users"))
				continue
			}

			for _, user := range users {
				if triggerUID > 0 && triggerUID != user.ID {
					continue
				}
				err := sendPushNotificationToUser(cfg, db, user)
				if err != nil {
					logrus.Error(errors.Wrap(err, "sending push notification"))
				}
			}
		}
	}()
	return nil
}

func getAllUsersWithSubscriptions(db *gorm.DB) ([]types.User, error) {
	var users []types.User
	err := db.Preload("PushSubscriptions").Find(&users).Error
	return users, err
}

func triggerPushes() echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if !ok {
			return c.String(http.StatusUnauthorized, "unauthorized")
		}
		if user.Role != "admin" {
			return c.String(http.StatusUnauthorized, "unauthorized, must be admin")
		}
		triggerPushChan <- user.ID
		fmt.Fprintln(c.Response().Writer, "Triggered pushes")
		return nil
	}
}

func sendPushNotificationToUser(cfg types.Config, db *gorm.DB, user types.User) error {
	logrus := logrus.WithField("user", user.Name)
	for _, subData := range user.PushSubscriptions {
		logrus := logrus.WithField("subdata", subData.ID)
		sub := &webpush.Subscription{
			Endpoint: subData.Endpoint,
			Keys: webpush.Keys{
				P256dh: subData.P256DH,
				Auth:   subData.Auth,
			},
		}

		prompt := randomPrompt()
		pushPayload, err := json.Marshal(map[string]string{
			"title": "Fanks",
			"body":  prompt,
			"icon":  fmt.Sprintf("https://%s/static/icon-192.png", cfg.Hostname),
			"badge": fmt.Sprintf("https://%s/static/badge-128.png", cfg.Hostname),
			"url":   fmt.Sprintf("/?prompt=%s", url.QueryEscape(prompt)),
		})
		if err != nil {
			return errors.Wrap(err, "marshalling push payload")
		}

		logrus.Debugf("sending push notification: %s", string(pushPayload))
		resp, err := webpush.SendNotification(pushPayload, sub, &webpush.Options{
			VAPIDPublicKey:  cfg.VapidPublicKey,
			VAPIDPrivateKey: cfg.VapidPrivateKey,
			TTL:             600,
			Urgency:         webpush.UrgencyVeryLow,
		})
		if err != nil {
			return errors.Wrap(err, "sending push notification")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 201 {
			var deleteErr error
			if resp.StatusCode == 410 {
				logrus.Info("Subscriber no longer active")
				deleteErr = db.Delete(subData).Error
			}
			return errs.Join(fmt.Errorf("Got status code %d", resp.StatusCode), deleteErr)
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Reading response body from push notifications")
		}

		logrus.Debugf("Got resp body (%d): %s", resp.StatusCode, string(respBody))

		logrus.Info("Sent push notification to user")
	}
	return nil
}

func removeSubscription(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if !ok {
			return c.String(http.StatusUnauthorized, "unauthorized")
		}

		if err := db.Delete(user.PushSubscriptions).Error; err != nil {
			return errors.Wrap(err, "removing subscription")
		}

		return c.String(http.StatusOK, "subscription removed")
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
