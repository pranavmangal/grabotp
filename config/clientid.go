package config

func ReadClientId() (string, error) {
	conf, err := ReadConfig()
	if err != nil {
		return "", err
	}

	return conf.ClientId, nil
}

func WriteClientId(clientId string) error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	conf.ClientId = clientId
	return WriteConfig(conf)
}

func DeleteClientId() error {
	conf, err := ReadConfig()
	if err != nil {
		return err
	}

	conf.ClientId = ""
	return WriteConfig(conf)
}
