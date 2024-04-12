package cache

import "time"

type Disabled struct{}

func NewDisabled() Disabled {
	return Disabled{}
}

func (Disabled) Get(_ string, _ bool, _ time.Duration) (any, bool, error) {
	return nil, false, nil
}

func (Disabled) Set(_ string, _ any, _ time.Duration) error {
	return nil
}
