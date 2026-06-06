package config

import "github.com/buyahref/payment-hub/internal/phonepe"

func (c *Config) PhonePeClientConfig() phonepe.Config {
	return phonepe.Config{
		MerchantID: c.PhonePeMerchantID,
		SaltKey:    c.PhonePeSaltKey,
		SaltIndex:  c.PhonePeSaltIndex,
		BaseURL:    phonepe.BaseURL(c.PhonePeEnv),
	}
}
