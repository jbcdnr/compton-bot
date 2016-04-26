package compton

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

// Transaction represents a purchase paid by someone for other people
type Transaction struct {
	Amount  float64   `bson:"amount"`
	PaidBy  string    `bson:"paid_by"`
	PaidFor []string  `bson:"paid_for"`
	Date    time.Time `bson:"timestamp"`
}

// Chat represents a money count for a group discussion
type Chat struct {
	ChatID       bson.ObjectId `bson:"chat_id"`
	People       []string      `bson:"people"`
	Transactions []Transaction `bson:"transactions"`
}
