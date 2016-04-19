package compton

import (
  "time"
)

// Transaction represents a purchase paid by someone for other people
type Transaction struct {
	Price   float64
	PaidBy  string
	PaidFor []string
	Date    time.Time
}