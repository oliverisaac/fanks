package types

import (
	errs "errors"
	"fmt"
	"net/mail"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/oliverisaac/goli"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AllowSignup       bool
	AllowSignupEmails []string
	CookeSecret       []byte
	DBPath            string
}

func ConfigFromEnv() (Config, error) {
	ret := Config{}
	var retErr error
	var err error

	ret.AllowSignup, err = strconv.ParseBool(goli.DefaultEnv("FANKS_ALLOW_SIGNUP", "false"))
	if err != nil {
		retErr = errs.Join(retErr, errors.Wrap(err, "parsing FANKS_ALLOW_SIGNUP"))
	}

	allowedEmails := strings.Split(os.Getenv("FANKS_ALLOW_SIGNUP_EMAILS"), ",")
	for _, e := range allowedEmails {
		if e == "" {
			continue
		}
		email, err := mail.ParseAddress(e)
		if err != nil {
			retErr = errs.Join(retErr, errors.Wrapf(err, "parsing email %q", e))
		} else {
			ret.AllowSignupEmails = append(ret.AllowSignupEmails, email.Address)
		}
	}
	logrus.Infof("Allowed signup emails: %v", ret.AllowSignupEmails)

	cookieSecret, ok := os.LookupEnv("FANKS_COOKIE_STORE_SECRET")
	if !ok {
		retErr = errs.Join(retErr, fmt.Errorf("You must define env FANKS_COOKIE_STORE_SECRET"))
	} else {
		ret.CookeSecret = []byte(cookieSecret)
	}

	ret.DBPath, ok = os.LookupEnv("FANKS_DB_PATH")
	if !ok {
		retErr = errs.Join(retErr, fmt.Errorf("You must define env FANKS_DB_PATH"))
	} else if _, err := os.Stat(path.Dir(ret.DBPath)); err != nil {
		retErr = errs.Join(retErr, errors.Wrap(err, "Directory for FANKS_DB_PATH must exist"))
	}

	return ret, retErr
}
