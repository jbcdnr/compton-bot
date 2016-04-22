package compton

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strings"
)

// HandleUpdate take care of the update
func HandleUpdate(update tgbotapi.Update, api *tgbotapi.BotAPI, db *mgo.Collection) {

	if update.Message != nil {

		chatID := update.Message.Chat.ID
		chatIDString := fmt.Sprintf("%d", chatID)

		_, err := addChatToDatabase(chatIDString, db)
		if err != nil {
			log.Fatal(err)
		}

		switch update.Message.Command() {
		case "addPurchase":
			promptText := "Who paid the expense ?"
			prompt := tgbotapi.NewMessage(update.Message.Chat.ID, promptText)
			sentPrompt, err := api.Send(prompt)

			if err != nil {
				log.Println(err)
				return
			}

			// retrieve the chat information (could project on People)
			chat := Chat{}
			err = db.Find(bson.M{"chat_id": chatIDString}).One(&chat)

			if err != nil {
				log.Printf("Did not find a chat: %s\n", err)
				addChatToDatabase(chatIDString, db)
				// TODO propose create new Tricount or just do it ?
				return
			}

			// propose to add people to tricount if nobody
			if len(chat.People) == 0 {
				pleaseAddPeople := "Nobody is registered for Compton in this chat, please run /addPeople"
				editMessage := tgbotapi.NewEditMessageText(update.Message.Chat.ID, sentPrompt.MessageID, pleaseAddPeople)
				api.Send(editMessage)
				return
			}

			// create the answer keyboard with everybody
			buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(chat.People))
			for _, people := range chat.People {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(people, "paidBy:"+people))
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
			keyboardUpdate := tgbotapi.NewEditMessageReplyMarkup(update.Message.Chat.ID, sentPrompt.MessageID, keyboard)

			api.Send(keyboardUpdate)

		case "addPeople":
			api.Send(newMessageForcedAnswer(chatID, "Please type the name of the first people to add..."))

		default:
			if update.Message.ReplyToMessage != nil { // got a reply to a previous message

				if update.Message.IsCommand() {
					if update.Message.Command() == "done" {
						people, _ := getPeopleInChat(chatIDString, db)
						all := strings.Join(people, ", ")
						// TODO better formating
						api.Send(tgbotapi.NewMessage(chatID, "We are done adding people. There are "+all))
					} else {
						api.Send(tgbotapi.NewMessage(chatID, "A name should not start with /"))
					}
					return
				}

				question := update.Message.ReplyToMessage.Text
				if question == "Please type the name of the first people to add..." ||
					strings.Contains(question, "ype another name or /done to finish") {
					peopleToAdd := update.Message.Text
					if peopleToAdd != "" {
						err := addPeopleToChat(peopleToAdd, chatIDString, db)
						if err == nil {
							answer := fmt.Sprintf("Added %s, type another name or /done to finish", peopleToAdd)
							api.Send(newMessageForcedAnswer(chatID, answer))
						} else {
							log.Printf("Error adding %s to chat %s: %s", peopleToAdd, chatIDString, err)
							api.Send(tgbotapi.NewMessage(chatID, "An error occured when adding the new person"))
						}
					} else {
						api.Send(tgbotapi.NewMessage(chatID, "We only accept non empty names, type another name or /done to finish"))
					}
				}
			}
		}
	}

	if update.CallbackQuery != nil {
		if strings.HasPrefix(update.CallbackQuery.Data, "paidBy:") {
			people := strings.TrimPrefix(update.CallbackQuery.Data, "paidBy:")
			_ = people
		}
		// TODO
		log.Printf("Received a callback: %+v\n", update.CallbackQuery)
		log.Printf("Received a callback message: %+v\n", update.CallbackQuery.Message)
	}
}

func newMessageForcedAnswer(chatID int64, text string) (message tgbotapi.MessageConfig) {
	message = tgbotapi.NewMessage(chatID, text)
	message.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
	return
}