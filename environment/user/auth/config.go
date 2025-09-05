package auth

import "obsessiontech/common/config"

var Config struct {
	SuperIP       string
	AuthTokenSalt string
}

var MockConfig struct {
	MockRegisterUserID int
	MockLoginUserID    int
}

func init() {
	config.GetConfig("config.yaml", &Config)
	config.GetConfig("mock_user_auth.yaml", &MockConfig)
}
