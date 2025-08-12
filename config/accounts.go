package config

import (
	"fmt"
	"slices"
)

func AddAccount(email string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	if slices.Contains(conf.Emails, email) {
		return nil
	}

	conf.Emails = append(conf.Emails, email)
	return WriteConfig(conf)
}

func ListAccounts() ([]string, error) {
	conf, err := ReadConfig()
	if err != nil {
		return nil, err
	}

	return conf.Emails, nil
}

func RemoveAccount(email string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	var newEmails []string
	found := false

	for _, e := range conf.Emails {
		if e != email {
			newEmails = append(newEmails, e)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Could not find %s", email)
	}

	conf.Emails = newEmails
	return WriteConfig(conf)
}

func ResetAllAccounts() error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	conf.Emails = []string{}
	return WriteConfig(conf)
}
