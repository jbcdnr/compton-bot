package compton

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func addChatToDatabase(chat int64, db *mgo.Collection) (info *mgo.ChangeInfo, err error) {
	emptyTransaction := []Transaction{}
	return db.Upsert(
		bson.M{"chat_id": chat},
		bson.M{"$setOnInsert": bson.M{"chat_id": chat, "people": []string{}, "transactions": emptyTransaction}})
}

func addPeopleToChat(people string, chat int64, db *mgo.Collection) (err error) {
	return db.Update(bson.M{"chat_id": chat}, bson.M{"$addToSet": bson.M{"people": people}})
}

func getPeopleInChat(chatID int64, db *mgo.Collection) (people []string, err error) {
	chatData := Chat{}
	err = db.Find(bson.M{"chat_id": chatID}).One(&chatData)
	return chatData.People, err
}

func addReplyAction(action ReplyAction, db *mgo.Collection) {
	db.Update(bson.M{"main": true}, bson.M{"$push": bson.M{"replies": action}})
}

func addCallbackAction(action CallbackAction, db *mgo.Collection) error {
	return db.Update(bson.M{"main": true}, bson.M{"$push": bson.M{"callbacks": action}})
}

func addTransaction(chatID int64, transaction Transaction, db *mgo.Collection) error {
	return db.Update(bson.M{"chat_id": chatID}, bson.M{ "$push": bson.M{"transactions": transaction}})
}
