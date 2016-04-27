package compton

import (
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strings"
	"strconv"
)

// HandleUpdate take care of the update
func HandleUpdate(update tgbotapi.Update, api *tgbotapi.BotAPI, db *mgo.Collection) {

	if update.Message != nil {

		if update.Message.ReplyToMessage != nil {
			err := onReply(*update.Message, api, db) // TODO care about err ?
			if err != nil {
				log.Println(err)
			}
			return
		}

		chatID := update.Message.Chat.ID

		_, err := addChatToDatabase(chatID, db)
		if err != nil {
			log.Fatal(err)
		}

		switch update.Message.Command() {
		case "balance":
			balance, err := balanceForChat(chatID, db)
			if err != nil {
				api.Send(tgbotapi.NewMessage(chatID, "Could not find a compton in this chat"))
				log.Println(err)
				return
			}
			
			strs := make([]string, 0, len(balance))
			for people, bal := range balance {
				strs = append(strs, fmt.Sprintf("- %s: %.2f$", people, bal))
			}
			message := strings.Join(strs, "\n")
			api.Send(tgbotapi.NewMessage(chatID, message))
		
		case "solve":
			balance, err := balanceForChat(chatID, db)
			if err != nil {
				api.Send(tgbotapi.NewMessage(chatID, "Could not find a compton in this chat"))
				log.Println(err)
				return
			}
			reimbursment := findOptimalArrangment(balance)
			strs := make([]string, 0, 20)
			for giver, pairs := range reimbursment {
				for _, pair := range pairs {
					strs = append(strs, fmt.Sprintf("- %s gives %.2f$ to %s", giver, pair.Amount, pair.People))
				}
			}
			api.Send(tgbotapi.NewMessage(chatID, strings.Join(strs, "\n")))
		
		case "addPurchase":

			// retrieve the chat information from DB or create it
			chat := Chat{}
			err = db.Find(bson.M{"chat_id": chatID}).One(&chat)
			if err != nil {
				addChatToDatabase(chatID, db)
				err = db.Find(bson.M{"chat_id": chatID}).One(&chat)
				if err != nil {
					log.Fatal(err)
				}
			}

			// propose to add people to tricount if nobody
			if len(chat.People) == 0 {
				pleaseAddPeople := "Nobody is registered for Compton in this chat, please run /addPeople"
				api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, pleaseAddPeople))
				return
			}

			promptText := "Who paid the expense ?"
			prompt := tgbotapi.NewMessage(update.Message.Chat.ID, promptText)
			sentPrompt, err := api.Send(prompt)
			if err != nil {
				log.Println(err)
				return
			}

			// create the answer keyboard with everybody
			buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(chat.People))
			for _, people := range chat.People {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(people, fmt.Sprintf("%d/%s", sentPrompt.MessageID, people)))
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
			keyboardUpdate := tgbotapi.NewEditMessageReplyMarkup(update.Message.Chat.ID, sentPrompt.MessageID, keyboard)

			api.Send(keyboardUpdate)

			action := CallbackAction{ChatID: chatID, Action: "paidBy", MessageID: sentPrompt.MessageID}
			err = addCallbackAction(action, db)
			if err != nil {
				log.Println(err)
			}

		case "addPeople":
			sendPeoplePrompt(*update.Message, api, db)
		default:
			// TODO sorry did not understand
		}
	}

	if update.CallbackQuery != nil {
		err := onCallback(*update.CallbackQuery, api, db)
		if err != nil {
			log.Println(err)
		}
	}
}

func newMessageForcedAnswer(chatID int64, text string) (message tgbotapi.MessageConfig) {
	message = tgbotapi.NewMessage(chatID, text)
	message.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
	return
}

func onReply(message tgbotapi.Message, api *tgbotapi.BotAPI, db *mgo.Collection) (err error) {
	questionID := message.ReplyToMessage.MessageID
	callbacks := CallbacksHandler{}

	// TODO should remove old object
	err = db.Find(bson.M{"main": true, "replies.message_id": questionID}).Select(bson.M{"replies.$": true}).One(&callbacks)
	if err != nil {
		log.Println(err)
		return
	}
	if len(callbacks.Replies) != 1 {
		return errors.New("Could not find the associated reply message in DB")
	}
	replyCallback := callbacks.Replies[0]
	db.Update(bson.M{"main": true, "replies.message_id": message.MessageID}, bson.M{"$pull": bson.M{ "replies": bson.M{"message_id": message.MessageID}}})

	switch replyCallback.Action {
	case "addPeople":
		onNewPeople(replyCallback, message, api, db)
	case "inputAmount":
		onAmountInput(replyCallback, message, api, db)
	}

	return
}

func onCallback(query tgbotapi.CallbackQuery, api *tgbotapi.BotAPI, db *mgo.Collection) (err error) {
	
	splits := strings.SplitN(query.Data, "/", 2)
	if len(splits) != 2 {
		return errors.New("Bad format for callback id and args")
	}
	callbackID, err := strconv.Atoi(splits[0])
	if err != nil {
		return err
	}
	arg := splits[1]

	// TODO should remove the object from the DB and clean
	callbacks := CallbacksHandler{}
	err = db.Find(bson.M{"main": true, "callbacks.message_id": callbackID}).Select(bson.M{"callbacks.$": true}).One(&callbacks)
	if err != nil {
		return
	}
	if len(callbacks.Callbacks) != 1 {
		return errors.New("Could not find the associated callback in DB")
	}
	callback := callbacks.Callbacks[0]
	db.Update(bson.M{"main": true, "callbacks.message_id": callbackID}, bson.M{"$pull": bson.M{ "callbacks": bson.M{"message_id": callbackID}}})

	switch callback.Action {
	case "paidBy":
		onPaidBy(callback, arg, api, db)
	case "addPaidFor":
		onAddPaidFor(callback, arg, api, db)
	}

	return
}

func onNewPeople(action ReplyAction, message tgbotapi.Message, api *tgbotapi.BotAPI, db *mgo.Collection) {
	chatID := message.Chat.ID

	if message.IsCommand() {
		if message.Command() == "done" {
			people, _ := getPeopleInChat(chatID, db)
			all := strings.Join(people, "\n- ")
			// TODO better formating
			api.Send(tgbotapi.NewMessage(message.Chat.ID, "We are done adding people. Here is a list: \n- "+all))
		} else {
			api.Send(tgbotapi.NewMessage(message.Chat.ID, "A name should not start with /"))
		}
		return
	}

	peopleToAdd := message.Text
	if peopleToAdd != "" {
		err := addPeopleToChat(peopleToAdd, chatID, db)
		if err == nil {
			answer := fmt.Sprintf("Added %s to the Compton", peopleToAdd)
			api.Send(tgbotapi.NewMessage(message.Chat.ID, answer))
			sendPeoplePrompt(message, api, db)
		} else {
			log.Printf("Error adding %s to chat %d: %s", peopleToAdd, chatID, err)
			api.Send(tgbotapi.NewMessage(message.Chat.ID, "An error occured when adding the new person"))
		}
	} else {
		api.Send(tgbotapi.NewMessage(message.Chat.ID, "We only accept non empty names"))
		sendPeoplePrompt(message, api, db)
	}

}

func onAmountInput(action ReplyAction, message tgbotapi.Message, api *tgbotapi.BotAPI, db *mgo.Collection) {
	amount, err := strconv.ParseFloat(message.Text, 64)
	if err != nil {
		log.Printf("Error parsing amount: %s", err)
		api.Send(tgbotapi.NewMessage(message.Chat.ID, "I did not understand the amount."))
	}
	
	transaction := action.Transaction
	transaction.Amount = amount
	promptPaidFor(0, transaction, message.Chat.ID, api, db)
}

func onPaidBy(callback CallbackAction, data string, api *tgbotapi.BotAPI, db *mgo.Collection) {
	people, _ := getPeopleInChat(callback.ChatID, db)
	correctPeople := false
	for _, p := range people {
		if p == data {
			correctPeople = true
			break
		}
	}
	if !correctPeople {
		api.Send(tgbotapi.NewMessage(callback.ChatID, "Unknown people"))
		return
	} 

	sent, _ := api.Send(newMessageForcedAnswer(callback.ChatID, fmt.Sprintf("How much did %s pay ?", data)))
	transaction := callback.Transaction
	transaction.PaidBy = data
	action := ReplyAction{MessageID: sent.MessageID, Transaction: transaction, Action: "inputAmount"}
	addReplyAction(action, db)
}

// TODO should always edit the same message
func onAddPaidFor(callback CallbackAction, data string, api *tgbotapi.BotAPI, db *mgo.Collection) {
	if data == "Done" {
		err := addTransaction(callback.ChatID, callback.Transaction, db)
		if err == nil {
			api.Send(tgbotapi.NewMessage(callback.ChatID, "Added the new transaction"))
		}
	} else if data == "All" {
		// retrieve the chat
		chat := Chat{}
		err := db.Find(bson.M{"chat_id": callback.ChatID}).One(&chat)
		if err != nil {
			log.Println(err)
		}
		
		transaction := callback.Transaction
		transaction.PaidFor = chat.People
		
		err = addTransaction(callback.ChatID, transaction, db)
		if err == nil {
			api.Send(tgbotapi.NewMessage(callback.ChatID, "Added the new transaction")) // TODO pretty print
		}
	} else {
		transaction := callback.Transaction
		transaction.PaidFor = append(transaction.PaidFor, data) // TODO check contained
		promptPaidFor(callback.MessageID, transaction, callback.ChatID, api, db)
	}
}

func promptPaidFor(promptID int, transaction Transaction, chatID int64, api *tgbotapi.BotAPI, db *mgo.Collection) {
	
	// retrieve the chat
	chat := Chat{}
	err := db.Find(bson.M{"chat_id": chatID}).One(&chat)
	if err != nil {
		log.Println(err)
	}
	
	addedSoFar := strings.Join(transaction.PaidFor, ", ")
	if addedSoFar != "" {
		addedSoFar = addedSoFar + "..."
	}
	text := fmt.Sprintf("Who did %s pay for ? %s", transaction.PaidBy, addedSoFar)
	if promptID == 0 {
		sent, _ := api.Send(tgbotapi.NewMessage(chatID, text))
		promptID = sent.MessageID
	} else {
		api.Send(tgbotapi.NewEditMessageText(chatID, promptID, text))
	}
	
	// create the answer keyboard with only new ones
	buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(chat.People) + 1)
	for _, people := range chat.People {
		contained := false
		for _, p := range transaction.PaidFor {
			if p == people {
				contained = true
				break
			}
		}
		if ! contained {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(people, fmt.Sprintf("%d/%s", promptID, people)))
		}
	}
	extra := "Done"
	if len(transaction.PaidFor) == 0 {
		extra = "All"
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(extra, fmt.Sprintf("%d/%s", promptID, extra)))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
	log.Println(keyboard)
	keyboardUpdate := tgbotapi.NewEditMessageReplyMarkup(chatID, promptID, keyboard)

	api.Send(keyboardUpdate)
	
	newAction := CallbackAction{Action: "addPaidFor", MessageID: promptID, ChatID: chat.ChatID, Transaction: transaction}
	addCallbackAction(newAction, db)
}

func sendPeoplePrompt(message tgbotapi.Message, api *tgbotapi.BotAPI, db *mgo.Collection) {
	sent, _ := api.Send(newMessageForcedAnswer(message.Chat.ID, "Type the name of the people to add or /done"))
	action := ReplyAction{MessageID: sent.MessageID, Action: "addPeople"}
	addReplyAction(action, db)
}

func balanceForChat(chatID int64, db *mgo.Collection) (balances map[string]float64, err error) {
	balances = make(map[string]float64)
	
	// retrieve the chat
	chat := Chat{}
	err = db.Find(bson.M{"chat_id": chatID}).One(&chat)
	if err != nil {
		return
	}
	
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