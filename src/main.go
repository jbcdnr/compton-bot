package main

import (
	"./compton"
	"bitbucket.org/mrd0ll4r/tbotapi"
	"fmt"
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
	api, err := tbotapi.New(apiToken)
	if err != nil {
		log.Fatal(err)
	}

	// just to show its working
	log.Printf("User ID: %d\n", api.ID)
	log.Printf("Bot Name: %s\n", api.Name)
	log.Printf("Bot Username: %s\n", api.Username)

	closed := make(chan struct{})
	wg := &sync.WaitGroup{}

	updateHandler := func() {
		defer wg.Done()
		for {
			select {
			case <-closed:
				return
			case update := <-api.Updates:
				if update.Error() != nil {
					log.Printf("Update error: %s\n", update.Error())
					continue
				}

				compton.HandleUpdate(update.Update(), api)
			}
		}
	}

  // run concurent workers to handle Updates
	NUMBER_HANDLERS := 4
	wg.Add(NUMBER_HANDLERS)
	for i := 0; i < NUMBER_HANDLERS; i++ {
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
	api.Close() // this might take a while
	close(closed)
	wg.Wait()
}
