package clock

import "time"

type System struct{}

func New() System {
	return System{}
}

func (System) Now() time.Time {
	return time.Now()
}
