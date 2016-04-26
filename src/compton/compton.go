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

/*
	callbacks in database with date (to clean)
	- answerMessage: messageId -> action with (ChatID, Transaction?, Message)
	- callback: callbackdata (id/args) -> action with (ChatID, Transaction?, data-args)
		* get people who paid
		* amount paid
		* new paid
		* done
*/

func onReply(message tgbotapi.Message, api *tgbotapi.BotAPI, db *mgo.Collection) (err error) {
	questionID := message.ReplyToMessage.MessageID
	callbacks := CallbacksHandler{}

	// TODO should remove the object from the DB and clean
	err = db.Find(bson.M{"main": true, "replies.message_id": questionID}).Select(bson.M{"replies.$": true}).One(&callbacks)
	if err != nil {
		log.Println(err)
		return
	}
	if len(callbacks.Replies) != 1 {
		return errors.New("Could not find the associated reply message in DB")
	}
	replyCallback := callbacks.Replies[0]

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
		// TODO	
	}
	_ = amount
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

func onAddPaidFor(callback CallbackAction, data string, api *tgbotapi.BotAPI, db *mgo.Collection) {

}

func sendPeoplePrompt(message tgbotapi.Message, api *tgbotapi.BotAPI, db *mgo.Collection) {
	sent, _ := api.Send(newMessageForcedAnswer(message.Chat.ID, "Type the name of the people to add or /done"))
	action := ReplyAction{MessageID: sent.MessageID, Action: "addPeople"}
	addReplyAction(action, db)
}
