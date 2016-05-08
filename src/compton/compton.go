package compton

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strconv"
	"strings"
)

// HandleUpdate take care of the update
func HandleUpdate(update tgbotapi.Update, api *tgbotapi.BotAPI, db *mgo.Collection) {

	if update.Message != nil {

		message := update.Message
		chatID := message.Chat.ID
		userID := message.From.ID

		// retrieve the chat information from DB or create it
		chatData := Chat{}
		err := db.Find(bson.M{"chat_id": chatID}).One(&chatData)
		if err != nil {
			empty := Chat{}
			empty.ChatID = chatID
			db.Upsert(
				bson.M{"chat_id": chatID},
				bson.M{"$setOnInsert": empty})
			err = db.Find(bson.M{"chat_id": chatID}).One(&chatData)
			if err != nil {
				log.Fatal(err)
			}
		}

		// handle direct init command
		if message.IsCommand() {

			switch message.Command() {

			case "balance":
				balance := chatData.balance()

				strs := make([]string, 0, len(balance))
				for people, bal := range balance {
					strs = append(strs, fmt.Sprintf("- %s: %.2f$", people, bal))
				}
				message := strings.Join(strs, "\n")
				api.Send(tgbotapi.NewMessage(chatID, message))
				return

			case "solve":
				balance := chatData.balance()
				reimbursment := findOptimalArrangment(balance)
				strs := make([]string, 0, 20)
				for giver, pairs := range reimbursment {
					for _, pair := range pairs {
						strs = append(strs, fmt.Sprintf("- %s gives %.2f$ to %s", giver, pair.Amount, pair.People))
					}
				}
				api.Send(tgbotapi.NewMessage(chatID, strings.Join(strs, "\n")))
				return

			case "whoShouldPay":
				balance := chatData.balance()

				shouldPay := ""
				smallest := -1.0
				for people, bal := range balance {
					if shouldPay == "" || bal < smallest {
						shouldPay = people
						smallest = bal
					}
				}

				api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("%s should pay.", shouldPay)))
				return

			case "list":

				strs := make([]string, len(chatData.Transactions))
				for i, t := range chatData.Transactions {
					strs[i] = fmt.Sprintf("%d. %s", i+1, t)
				}
				api.Send(tgbotapi.NewMessage(chatID, strings.Join(strs, "\n")))

			case "addPurchase":

				promptText := "Who paid the expense ?"
				prompt := tgbotapi.NewMessage(update.Message.Chat.ID, promptText)

				// create the answer keyboard with everybody
				buttons := make([]tgbotapi.KeyboardButton, 0, len(chatData.People))
				for _, people := range chatData.People {
					buttons = append(buttons, tgbotapi.NewKeyboardButton(people))
				}
				keyboard := tgbotapi.NewReplyKeyboard(buttons)
				prompt.ReplyMarkup = keyboard

				_, err := api.Send(prompt)

				if err == nil {
					interaction := Interaction{}
					interaction.Author = userID
					interaction.Type = "addPurchase/paidBy"
					interaction.Transaction = &Transaction{}
					addInteractionToChat(interaction, chatID, db)
				}

				return

			case "addPeople":

				log.Println("addPeople received")
				api.Send(tgbotapi.NewMessage(chatID, "Type the name of a person to add or /done."))
				interaction := Interaction{}
				interaction.Author = userID
				interaction.Type = "addPeople"
				addInteractionToChat(interaction, chatID, db)
				return

			}
		}

		// handle interactions
		var interaction *Interaction
		for _, inter := range chatData.Interactions {
			if inter.Author == userID {
				interaction = &inter
				break
			}
		}
		if interaction == nil {
			log.Printf("User replied to no question")
			// TODO message failed
		}

		switch interaction.Type {
		case "addPeople":
			if message.IsCommand() && message.Command() == "done" {
				if len(chatData.People) == 0 {
					api.Send(tgbotapi.NewMessage(chatID, "The list of people in the compton is empty"))
				} else {
					list := chatData.People
					for i, p := range list {
						list[i] = "- " + p
					}
					all := strings.Join(list, "\n")
					api.Send(tgbotapi.NewMessage(chatID, "The list of people in the compton is:\n"+all))
				}

				removeInteractionsForUser(chatID, userID, db)

				return
			}

			people := message.Text
			if people == "" {
				api.Send(tgbotapi.NewMessage(chatID, "The name must be non empty"))
			} else {
				addPeopleToChat(people, chatID, db)
			}

			api.Send(tgbotapi.NewMessage(chatID, "Type the name of another person to add or /done."))

		case "addPurchase/paidBy":

			people := message.Text
			contained := false
			for _, p := range chatData.People {
				if p == people {
					contained = true
					break
				}
			}

			if !contained {
				// TODO error message
				return
			}

			db.Update(bson.M{"chat_id": chatID, "interactions.author": userID}, bson.M{"$set": bson.M{
				"interactions.$.transaction.paid_by": people,
				"interactions.$.type":                "addPurchase/amount"}})

			mes := tgbotapi.NewMessage(chatID, "How much did "+people+" pay ?")
			mes.ReplyMarkup = tgbotapi.NewHideKeyboard(true)
			api.Send(mes)

		case "addPurchase/amount":
			amount, err := strconv.ParseFloat(message.Text, 64)
			if err != nil {
				log.Printf("Parse error: %s\n", err)
				// TODO handle error
			}

			mes := tgbotapi.NewMessage(chatID, "Who did "+interaction.Transaction.PaidBy+" pay for ?")
			mes.ReplyMarkup = keyboardWithPeople(chatData.People, interaction.Transaction)
			sent, err := api.Send(mes)

			if err != nil {
				// TODO handle error
			}

			db.Update(bson.M{"chat_id": chatID, "interactions.author": userID}, bson.M{"$set": bson.M{
				"interactions.$.transaction.amount": amount,
				"interactions.$.type":               "addPurchase/paidFor",
				"interactions.$.last_message":       sent.MessageID}})

		case "addPurchase/paidFor":

			if message.IsCommand() {
				switch message.Command() {
				case "all":
					interaction.Transaction.PaidFor = chatData.People
					fallthrough
				case "done":
					if len(interaction.Transaction.PaidFor) == 0 {
						// TODO error
					}
					addTransaction(chatID, *interaction.Transaction, db)
					mes := tgbotapi.NewMessage(chatID, (*interaction.Transaction).String())
					mes.ReplyMarkup = tgbotapi.NewHideKeyboard(true)
					api.Send(mes)
					return
				default:
					// TODO error
					return
				}
			}

			people := message.Text
			contained := false
			for _, p := range chatData.People {
				if p == people {
					contained = true
					break
				}
			}

			if !contained {
				// TODO error message
				return
			}

			db.Update(bson.M{"chat_id": chatID, "interactions.author": userID}, bson.M{"$addToSet": bson.M{
				"interactions.$.transaction.paid_for": people}})
			interaction.Transaction.PaidFor = append(interaction.Transaction.PaidFor, people)
			keyboard := keyboardWithPeople(chatData.People, interaction.Transaction)
			// edit := tgbotapi.NewEditMessageReplyMarkup(chatID, interaction.LastMessage, keyboard)

			mes := tgbotapi.NewMessage(chatID, "Who did "+interaction.Transaction.PaidBy+" pay for ?")
			mes.ReplyMarkup = keyboard
			api.Send(mes)

		}

	}
}

// transaction != nil, will take only new, All and /done (if non empty)
func keyboardWithPeople(people []string, transaction *Transaction) tgbotapi.ReplyKeyboardMarkup {

	alreadyPicked := func(pp string) bool {
		if transaction == nil {
			return false
		}
		for _, p := range transaction.PaidFor {
			if p == pp {
				return true
			}
		}
		return false

	}

	// create the answer keyboard with everybody
	buttons := make([]tgbotapi.KeyboardButton, 0, len(people)+1)
	for _, p := range people {
		check := ""
		if alreadyPicked(p) {
			check = "\xE2\x9C\x85 "
		}
		buttons = append(buttons, tgbotapi.NewKeyboardButton(check+p))
	}

	if transaction != nil {
		buttons = append([]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton("/all")}, buttons...)
		if len(transaction.PaidFor) != 0 {
			buttons = append(buttons, tgbotapi.NewKeyboardButton("/done"))
		}
	}
	keyboard := tgbotapi.NewReplyKeyboard(buttons)

	return keyboard
}
