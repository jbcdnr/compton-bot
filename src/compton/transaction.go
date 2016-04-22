package compton

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

// Transaction represents a purchase paid by someone for other people
type Transaction struct {
	ID      bson.ObjectId `bson:"_id,omitempty"`
	Amount  float64       `bson:"amount"`
	Author  string        `bson:"author"`
	PaidBy  string        `bson:"paid_by"`
	PaidFor []string      `bson:"paid_for"`
	Date    time.Time     `bson:"timestamp"`
}

// BuildingTransaction is a stage of build of Transaction
type BuildingTransaction struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	CallbackID  string        `bson:"callback_id"`
	Stage       string        `bson:"stage"`
	Transaction Transaction   `bson:"transaction"`
	Date        time.Time     `bson:"timestamp"`
}

// Chat represents a money count for a group discussion
type Chat struct {
	ID              bson.ObjectId `bson:"_id,omitempty"`
	ChatID          bson.ObjectId `bson:"chat_id"`
	People          []string      `bson:"people"`
	NewTransactions []Transaction `bson:"new_transactions"`
	Transactions    []Transaction `bson:"transactions"`
}
