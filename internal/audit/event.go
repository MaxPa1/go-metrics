package audit

import "time"

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

func NewEvent(metrics []string, ip string) Event {
	return Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ip,
	}
}
