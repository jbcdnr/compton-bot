package main

import (
	"./compton"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/mgo.v2"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {

	log.Printf("Starting the bot...\n")

	// get the bot API token from file apiToken
	buf, err := ioutil.ReadFile("apiToken")
	if err != nil {
		log.Fatal("No API token file found")
	}
	apiToken := strings.TrimSpace(string(buf))
	log.Printf("API token: '%s'\n", apiToken)

	// start the API connection
	api, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		log.Panic(err)
	}
	api.Debug = true

	// just to show its working
	log.Printf("User ID: %d\n", api.Self.ID)
	log.Printf("Bot Name: %s\n", api.Self.FirstName)
	log.Printf("Bot Username: %s\n", api.Self.UserName)

	config := tgbotapi.NewUpdate(0)
	config.Timeout = 60
	updatesChanel, err := api.GetUpdatesChan(config)

	if err != nil {
		log.Fatal(err)
	}

	// connect to MongoDB

	mongoSession, err := mgo.Dial("localhost:27017")
	if err != nil {
		log.Fatal(err)
	}
	defer mongoSession.Close()

	mongoSession.SetMode(mgo.Monotonic, true)

	closed := make(chan struct{})
	wg := &sync.WaitGroup{}

	updateHandler := func() {
		defer wg.Done()
		mongo := mongoSession.Copy().DB("test")
		for {
			select {
			case <-closed:
				return
			case update := <-updatesChanel:
				compton.HandleUpdate(update, api, mongo)
			}
		}
	}

	// run concurent workers to handle Updates
	numberHandler := 1
	wg.Add(numberHandler)
	for i := 0; i < numberHandler; i++ {
		go updateHandler()
	}

	// ensure a clean shutdown
	closing := make(chan struct{})
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdown
		signal.Stop(shutdown)
		close(shutdown)
		close(closing)
	}()

	fmt.Println("Bot started. Press CTRL-C to close...")

	// wait for the signal
	<-closing
	fmt.Println("Closing...")

	// always close the API first, let it clean up the update loop
	close(closed)
	wg.Wait()
}
