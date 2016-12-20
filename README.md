# Compton Telegram Bot

Implementation of a telegram bot in go-lang following the idea of
[Tricount app](https://www.tricount.com/en/).

Basic features include:

- Multiple user groups
- Interactive input
- Multiple currencies
- Solve the debt/credit problem

### Installation

- You need an API token from Telegram, place it in `/apiToken` file
- A [MongoDB database](https://www.mongodb.com/) to handle the data on the server
- Install the missing go dependences `cd src ; go get -d .`
