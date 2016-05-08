package compton

import (
	"fmt"
	"strings"
	"time"
)

// Transaction represents a purchase paid by someone for other people
type Transaction struct {
	Amount  float64   `bson:"amount"`
	PaidBy  string    `bson:"paid_by"`
	PaidFor []string  `bson:"paid_for"`
	Date    time.Time `bson:"timestamp"`
}

func (t Transaction) String() string {
	if len(t.PaidFor) == 0 {
		// TODO error
		return ""
	}

	all := strings.Join(t.PaidFor[0:len(t.PaidFor)-1], ", ")
	if len(t.PaidFor) > 1 {
		all += " and "
	}
	all += t.PaidFor[len(t.PaidFor)-1]

	return fmt.Sprintf("%s paid %.2f$ for %s", t.PaidBy, t.Amount, all)
}

// Chat represents a money count for a group discussion
type Chat struct {
	ChatID       int64         `bson:"chat_id"`
	People       []string      `bson:"people"`
	Transactions []Transaction `bson:"transactions"`
	Interactions []Interaction `bson:"interactions"`
}

type Interaction struct {
	Author      int          `bson:"author"`
	Type        string       `bson:"type"`
	Transaction *Transaction `bson:"transaction"`
	LastMessage int          `bson:"last_message"`
}
