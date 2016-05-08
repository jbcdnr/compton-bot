package compton

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func addPeopleToChat(people string, chat int64, db *mgo.Collection) (err error) {
	return db.Update(bson.M{"chat_id": chat}, bson.M{"$addToSet": bson.M{"people": people}})
}

func addInteractionToChat(interaction Interaction, chatID int64, db *mgo.Collection) (err error) {
	// TODO better upsert ?
	removeInteractionsForUser(chatID, interaction.Author, db)
	_, err = db.Upsert(bson.M{"chat_id": chatID}, bson.M{"$push": bson.M{"interactions": interaction}})
	return
}

func removeInteractionsForUser(chatID int64, userID int, db *mgo.Collection) {
	db.Update(bson.M{"chat_id": chatID}, bson.M{"$pull": bson.M{"interactions": bson.M{"author": userID}}})
}

func getPeopleInChat(chatID int64, db *mgo.Collection) (people []string, err error) {
	chatData := Chat{}
	err = db.Find(bson.M{"chat_id": chatID}).One(&chatData)
	return chatData.People, err
}

func addTransaction(chatID int64, transaction Transaction, db *mgo.Collection) error {
	return db.Update(bson.M{"chat_id": chatID}, bson.M{"$push": bson.M{"transactions": transaction}})
}

func (chat Chat) balance() (balances map[string]float64) {
	balances = make(map[string]float64)

	for _, people := range chat.People {
		balances[people] = 0
	}

	for _, transaction := range chat.Transactions {
		balances[transaction.PaidBy] += transaction.Amount
		part := transaction.Amount / float64(len(transaction.PaidFor))
		for _, p := range transaction.PaidFor {
			balances[p] -= part
		}
	}

	return
}
