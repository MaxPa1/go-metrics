package audit

import "time"

// Event описывает событие аудита: время, имена метрик и IP клиента.
type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// NewEvent создаёт событие с текущим временем.
func NewEvent(metrics []string, ip string) Event {
	return Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ip,
	}
}
