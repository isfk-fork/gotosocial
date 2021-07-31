/*
   GoToSocial
   Copyright (C) 2021 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/superseriousbusiness/gotosocial/internal/cliactions"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/db/pg"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/util"
	"golang.org/x/crypto/bcrypt"
)

// Create creates a new account in the database using the provided flags.
var Create cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	dbConn, err := pg.NewPostgresService(ctx, c, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	username, ok := c.AccountCLIFlags[config.UsernameFlag]
	if !ok {
		return errors.New("no username set")
	}
	if err := util.ValidateUsername(username); err != nil {
		return err
	}

	email, ok := c.AccountCLIFlags[config.EmailFlag]
	if !ok {
		return errors.New("no email set")
	}
	if err := util.ValidateEmail(email); err != nil {
		return err
	}

	password, ok := c.AccountCLIFlags[config.PasswordFlag]
	if !ok {
		return errors.New("no password set")
	}
	if err := util.ValidateNewPassword(password); err != nil {
		return err
	}

	_, err = dbConn.NewSignup(username, "", false, email, password, nil, "", "", false, false)
	if err != nil {
		return err
	}

	return dbConn.Stop(ctx)
}

// Confirm sets a user to Approved, sets Email to the current UnconfirmedEmail value, and sets ConfirmedAt to now.
var Confirm cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	dbConn, err := pg.NewPostgresService(ctx, c, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	username, ok := c.AccountCLIFlags[config.UsernameFlag]
	if !ok {
		return errors.New("no username set")
	}
	if err := util.ValidateUsername(username); err != nil {
		return err
	}

	a := &gtsmodel.Account{}
	if err := dbConn.GetLocalAccountByUsername(username, a); err != nil {
		return err
	}

	u := &gtsmodel.User{}
	if err := dbConn.GetWhere([]db.Where{{Key: "account_id", Value: a.ID}}, u); err != nil {
		return err
	}

	u.Approved = true
	u.Email = u.UnconfirmedEmail
	u.ConfirmedAt = time.Now()
	if err := dbConn.UpdateByID(u.ID, u); err != nil {
		return err
	}

	return dbConn.Stop(ctx)
}

// Promote sets a user to admin.
var Promote cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	dbConn, err := pg.NewPostgresService(ctx, c, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	username, ok := c.AccountCLIFlags[config.UsernameFlag]
	if !ok {
		return errors.New("no username set")
	}
	if err := util.ValidateUsername(username); err != nil {
		return err
	}

	a := &gtsmodel.Account{}
	if err := dbConn.GetLocalAccountByUsername(username, a); err != nil {
		return err
	}

	u := &gtsmodel.User{}
	if err := dbConn.GetWhere([]db.Where{{Key: "account_id", Value: a.ID}}, u); err != nil {
		return err
	}
	u.Admin = true
	if err := dbConn.UpdateByID(u.ID, u); err != nil {
		return err
	}

	return dbConn.Stop(ctx)
}

// Demote sets admin on a user to false.
var Demote cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	dbConn, err := pg.NewPostgresService(ctx, c, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	username, ok := c.AccountCLIFlags[config.UsernameFlag]
	if !ok {
		return errors.New("no username set")
	}
	if err := util.ValidateUsername(username); err != nil {
		return err
	}

	a := &gtsmodel.Account{}
	if err := dbConn.GetLocalAccountByUsername(username, a); err != nil {
		return err
	}

	u := &gtsmodel.User{}
	if err := dbConn.GetWhere([]db.Where{{Key: "account_id", Value: a.ID}}, u); err != nil {
		return err
	}
	u.Admin = false
	if err := dbConn.UpdateByID(u.ID, u); err != nil {
		return err
	}

	return dbConn.Stop(ctx)
}

// Disable sets Disabled to true on a user.
var Disable cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	dbConn, err := pg.NewPostgresService(ctx, c, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	username, ok := c.AccountCLIFlags[config.UsernameFlag]
	if !ok {
		return errors.New("no username set")
	}
	if err := util.ValidateUsername(username); err != nil {
		return err
	}

	a := &gtsmodel.Account{}
	if err := dbConn.GetLocalAccountByUsername(username, a); err != nil {
		return err
	}

	u := &gtsmodel.User{}
	if err := dbConn.GetWhere([]db.Where{{Key: "account_id", Value: a.ID}}, u); err != nil {
		return err
	}
	u.Disabled = true
	if err := dbConn.UpdateByID(u.ID, u); err != nil {
		return err
	}

	return dbConn.Stop(ctx)
}

// Suspend suspends the target account, cleanly removing all of its media, followers, following, likes, statuses, etc.
var Suspend cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	// TODO
	return nil
}

// Password sets the password of target account.
var Password cliactions.GTSAction = func(ctx context.Context, c *config.Config, log *logrus.Logger) error {
	dbConn, err := pg.NewPostgresService(ctx, c, log)
	if err != nil {
		return fmt.Errorf("error creating dbservice: %s", err)
	}

	username, ok := c.AccountCLIFlags[config.UsernameFlag]
	if !ok {
		return errors.New("no username set")
	}
	if err := util.ValidateUsername(username); err != nil {
		return err
	}

	password, ok := c.AccountCLIFlags[config.PasswordFlag]
	if !ok {
		return errors.New("no password set")
	}
	if err := util.ValidateNewPassword(password); err != nil {
		return err
	}

	a := &gtsmodel.Account{}
	if err := dbConn.GetLocalAccountByUsername(username, a); err != nil {
		return err
	}

	u := &gtsmodel.User{}
	if err := dbConn.GetWhere([]db.Where{{Key: "account_id", Value: a.ID}}, u); err != nil {
		return err
	}

	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %s", err)
	}

	u.EncryptedPassword = string(pw)

	if err := dbConn.UpdateByID(u.ID, u); err != nil {
		return err
	}

	return nil
}
