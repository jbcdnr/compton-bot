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

				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("%s should pay.", shouldPay))
				msg.ReplyToMessageID = message.MessageID
				api.Send(msg)
				return

			case "list":

				strs := make([]string, len(chatData.Transactions))
				for i, t := range chatData.Transactions {
					strs[i] = fmt.Sprintf("%d. %s", i+1, t)
				}
				text := strings.Join(strs, "\n")
				if len(chatData.Transactions) == 0 {
					text = "No transaction in the compton so far."
				}
				msg := tgbotapi.NewMessage(chatID, text)
				msg.ReplyToMessageID = message.MessageID
				api.Send(msg)
				return

			case "paid":

				promptText := "Who paid the expense ?"
				prompt := tgbotapi.NewMessage(update.Message.Chat.ID, promptText)
				prompt.ReplyToMessageID = message.MessageID

				// create the answer keyboard with everybody
				keyboard := keyboardWithPeople(chatData.People, nil)
				prompt.ReplyMarkup = keyboard

				mess, err := api.Send(prompt)

				if err == nil {
					interaction := Interaction{}
					interaction.Author = userID
					interaction.Type = "paid/paidBy"
					interaction.Transaction = &Transaction{}
					interaction.LastMessage = mess.MessageID
					addInteractionToChat(interaction, chatID, db)
				} else {
					// TODO error message
				}

				return

			case "addPeople":

				log.Println("addPeople received")
				prompt := tgbotapi.NewMessage(chatID, "Type the name of a person to add or /done.")
				prompt.ReplyToMessageID = message.MessageID
				prompt.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
				api.Send(prompt)
				interaction := Interaction{}
				interaction.Author = userID
				interaction.Type = "addPeople"
				addInteractionToChat(interaction, chatID, db)
				return
			default:

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
			return
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
			} else if people[0] == '/' {
				api.Send(tgbotapi.NewMessage(chatID, "The name must not start with '/'"))
			} else {
				addPeopleToChat(people, chatID, db)
			}

			prompt := tgbotapi.NewMessage(chatID, "Type the name of another person to add or /done.")
			prompt.ReplyToMessageID = message.MessageID
			prompt.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
			api.Send(prompt)

		case "paid/amount":
			amount, err := strconv.ParseFloat(message.Text, 64)
			if err != nil {
				log.Printf("Parse error: %s\n", err)
				return
				// TODO handle error
			}

			mes := tgbotapi.NewMessage(chatID, "Who did "+interaction.Transaction.PaidBy+" pay for ?")
			mes.ReplyToMessageID = message.MessageID
			keyboard := keyboardWithPeople(chatData.People, interaction.Transaction)
			mes.ReplyMarkup = keyboard
			sent, err := api.Send(mes)

			if err != nil {
				// TODO handle error
			}

			db.Update(bson.M{"chat_id": chatID, "interactions.author": userID}, bson.M{"$set": bson.M{
				"interactions.$.transaction.amount": amount,
				"interactions.$.type":               "paid/paidFor",
				"interactions.$.last_message":       sent.MessageID}})

		}

	}

	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {

		data := update.CallbackQuery.Data
		answerToMessage := update.CallbackQuery.Message

		chatID := answerToMessage.Chat.ID
		userID := update.CallbackQuery.From.ID

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

		_ = data
		_ = answerToMessage

		// handle interactions
		var interaction *Interaction
		for _, inter := range chatData.Interactions {
			if inter.LastMessage == answerToMessage.MessageID {
				interaction = &inter
				break
			}
		}
		if interaction == nil {
			log.Printf("User replied to no question")
			return
			// TODO message failed
		}
		
		switch interaction.Type {
		case "paid/paidBy":

			people := data
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
				"interactions.$.type":                "paid/amount"}})

			mes := tgbotapi.NewMessage(chatID, "How much did "+people+" pay ?")
			// mes.ReplyToMessageID = message.MessageID
			mes.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
			api.Send(mes)


		case "paid/paidFor":

				switch data {
				case "/all":
					interaction.Transaction.PaidFor = chatData.People
					fallthrough
				case "/done":
					if len(interaction.Transaction.PaidFor) == 0 {
						// TODO error
					}
					addTransaction(chatID, *interaction.Transaction, db)
					mes := tgbotapi.NewMessage(chatID, (*interaction.Transaction).String())
					mes.ReplyMarkup = tgbotapi.NewHideKeyboard(true)
					api.Send(mes)
					return
				default:
				}

			people := data
			delete := false

			if strings.HasPrefix(people, "\xE2\x9C\x85 ") {
				people = strings.TrimPrefix(people, "\xE2\x9C\x85 ")
				delete = true
			}

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

			if delete {
				db.Update(bson.M{"chat_id": chatID, "interactions.author": userID}, bson.M{"$pull": bson.M{
					"interactions.$.transaction.paid_for": people}})
				newPeople := make([]string, 0, len(interaction.Transaction.PaidFor))
				for _, p := range interaction.Transaction.PaidFor {
					if p != people {
						newPeople = append(newPeople, p)
					}
				}
				interaction.Transaction.PaidFor = newPeople
			} else {
				db.Update(bson.M{"chat_id": chatID, "interactions.author": userID}, bson.M{"$addToSet": bson.M{
					"interactions.$.transaction.paid_for": people}})
				interaction.Transaction.PaidFor = append(interaction.Transaction.PaidFor, people)
			}

			keyboard := keyboardWithPeople(chatData.People, interaction.Transaction)
			api.Send(tgbotapi.NewEditMessageReplyMarkup(chatID, answerToMessage.MessageID, keyboard))

	}
	}
}

// transaction != nil, will take only new, All and /done (if non empty)
func keyboardWithPeople(people []string, transaction *Transaction) tgbotapi.InlineKeyboardMarkup {

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

	createRowButton := func(str string) []tgbotapi.InlineKeyboardButton {
		return []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(str, str)}
	}

	// create the answer keyboard with everybody
	buttonRows := make([][]tgbotapi.InlineKeyboardButton, 0, len(people)+2)
	for _, p := range people {
		check := ""
		if alreadyPicked(p) {
			check = "\xE2\x9C\x85 "
		}
		buttonRows = append(buttonRows, createRowButton(check+p))
	}

	if transaction != nil {
		buttonRows = append([][]tgbotapi.InlineKeyboardButton{createRowButton("/all")}, buttonRows...)
		if len(transaction.PaidFor) != 0 {
			buttonRows = append(buttonRows, createRowButton("/done"))
		}
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttonRows...)

	return keyboard
}
