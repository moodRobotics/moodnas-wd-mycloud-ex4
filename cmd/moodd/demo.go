package main

import (
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

// Public demo credentials. They exist only with -simulate -demo; on real
// hardware the first administrator is always created by the user through the
// one-time setup code, and there is no default password.
const (
	DemoUser     = "demo"
	DemoPassword = "moodnas-demo"
)

var demoMode bool

func createDemoAccount(store *state.Store) error {
	if len(store.Read().Users) > 0 {
		return nil
	}
	hash, err := auth.HashPassword(DemoPassword)
	if err != nil {
		return err
	}
	return store.Update(func(d *state.Data) error {
		d.Users = append(d.Users, state.User{Name: DemoUser, Hash: hash, Role: state.RoleAdmin, Created: time.Now().UTC()})
		d.Shares = []state.Share{
			{Name: "Public", Users: []string{"*"}, Sentinel: true},
			{Name: "Family", Users: []string{"*"}, Sentinel: true},
			{Name: "Documents", Users: []string{DemoUser}, Sentinel: true},
		}
		return nil
	})
}
