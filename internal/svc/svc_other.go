//go:build !windows

package svc

import "time"

func List(string, bool) ([]Service, error)           { return nil, ErrUnsupported }
func Start(string, time.Duration) (Service, error)   { return Service{}, ErrUnsupported }
func Stop(string, time.Duration) (Service, error)    { return Service{}, ErrUnsupported }
func Restart(string, time.Duration) (Service, error) { return Service{}, ErrUnsupported }
