package compton

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func addChatToDatabase(chat string, db *mgo.Collection) (info *mgo.ChangeInfo, err error) {
	emptyTransaction := []Transaction{}
	return db.Upsert(
		bson.M{"chat_id": chat},
		bson.M{"$setOnInsert": bson.M{"chat_id": chat, "people": []string{}, "transactions": emptyTransaction}})
}

func addPeopleToChat(people, chat string, db *mgo.Collection) (err error) {
	return db.Update(bson.M{"chat_id": chat}, bson.M{"$addToSet": bson.M{"people": people}})
}

func getPeopleInChat(chat string, db *mgo.Collection) (people []string, err error) {
	chatData := Chat{}
	err = db.Find(bson.M{"chat_id": chat}).One(&chatData)
	return chatData.People, err
}
