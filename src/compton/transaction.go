package compton

import (
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
	ChatID       int64         `bson:"chat_id"`
	People       []string      `bson:"people"`
	Transactions []Transaction `bson:"transactions"`
}

type CallbacksHandler struct {
	Replies   []ReplyAction    `bson:"replies"`
	Callbacks []CallbackAction `bson:"callbacks"`
}

// TODO add timestamp for cleaning
type ReplyAction struct {
	MessageID   int         `bson:"message_id"`
	Action      string      `bson:"action"`
	Transaction Transaction `bson:"transaction"`
}

type CallbackAction struct {
	MessageID   int         `bson:"message_id"`
	Action      string      `bson:"action"`
	ChatID      int64       `bson:"chat_id"`
	Transaction Transaction `bson:"transaction"`
}
