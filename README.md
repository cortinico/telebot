# Telebot

A simple Telegram bot skeleton written in Go.

## Usage

You simply need a configuration (BotName + API Key) and a Response function.

Checkout this sample code:
```go
package main

import "github.com/cortinico/telebot"

func main() {
	conf := telebot.Configuration{
		BotName: "SampleBot",
		ApiKey:  "162227600:AAAAAAAAAAABBBBBBBBBBCCCCCCCCCDDDDD"}

	var bot telebot.Bot

	bot.Start(conf, func(mess string) (string, error) {
		var answer string
		switch mess {
		case "/test":
			answer = "Test command works :)"
		default:
			answer = "You typed " + mess
		}
		return answer, nil
	})
}
```

## Licence

The following software is released under the [MIT Licence](https://github.com/cortinico/telebot/blob/master/LICENSE)