package services

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type ServicesConfig func(*Services) error

func WithRequest() ServicesConfig {
	return func(s *Services) error {
		s.Request = NewRequestService(s)
		return nil
	}
}

// Initialize all services with all necessary parameters
func NewServices(cfgs ...ServicesConfig) (*Services, error) {
	var s Services
	for _, cfg := range cfgs {
		if err := cfg(&s); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

type Services struct {
	Request RequestService
}
