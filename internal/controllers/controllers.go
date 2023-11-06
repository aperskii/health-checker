package controllers

type ControllersConfig func(*Controllers) error

func WithAppController() ControllersConfig {
	return func(c *Controllers) error {
		c.AppController = NewAppController()
		return nil
	}
}

func NewControllers(cfgs ...ControllersConfig) (*Controllers, error) {
	var c Controllers
	for _, cfg := range cfgs {
		if err := cfg(&c); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

type Controllers struct {
	AppController *AppController
}
