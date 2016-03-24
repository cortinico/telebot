// Package for creating a running a simple Telegram bot.
// This bot is capable just to answer simple user/group messages,
// all the logic must be implemented inside a Responder func
package telebot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Interface Repserent a generic telegram bot. Exported functions
// are just LoadSettings to load a configuration and Start to
// launch the bot.
type IBot interface {
	Start(conf Configuration, resp Responder)
	LoadSettings(filename string) (Configuration, error)
	telegramPollURL(conf Configuration, offset string) string
	telegramSendURL(conf Configuration) string
	getTeleMessages(mess chan teleResults, conf Configuration)
	handleTeleMessages(infomes chan teleResults, conf Configuration)
	getResponse(message string, conf Configuration) string
}

// Struct representing a telegram Bot (will implement IBot).
// Bot has no field (no state), it's just an empty bot
type Bot struct{}

// Responder function, responsible of handling to user commands.
// This function represent the logic of your bot, you must provide
// a couple (string, error) for every message. The returned string
// will be sent to the user. If you set the error, the user will
// see an informative message.
type Responder func(string) (string, error)

// Configuration struct representing the configuration used from
// the bot to run properly. Configuration is usually loaded from file,
// or hardcoded inside the client code.
type Configuration struct {
	BotName string `json:"BotName"` // Name of the bot
	ApiKey  string `json:"ApiKey"`  // API Key of the bot (ask @BotFather)
	Timeout string `json:"Timeout"` // Timeout in seconds for polling
}

// Starts the telegram bot. The parameter conf represent the running
// configuration. The conf is mandatory otherwise the bot can't authenticate.
// The parameter resp is the Responder function. Also this parameter is
// mandatory, otherwise the bot don't know how to anser to user questions.
func (t Bot) Start(conf Configuration, resp Responder) {

	fmt.Println("INFO: Welcome to Telebot!")

	// Settings management
	if len(conf.ApiKey) == 0 {
		fmt.Println("FATAL: API Key not set. Please check your configuration")
		os.Exit(1)
	}
	if len(conf.BotName) == 0 {
		fmt.Println("FATAL: Bot Name not set. Please check your configuration")
		os.Exit(1)
	}
	fmt.Println("INFO: Settings loaded!")
	fmt.Println("INFO: Working as: " + conf.BotName)

	// Signal management
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Telegram messages channel
	mess := make(chan teleResults, 1)

	// Telegram Go-routines starts
	go t.getTeleMessages(mess, conf)
	go t.handleTeleMessages(mess, conf, resp)

	<-sigs
	fmt.Printf("INFO: Exiting...\n")

}

// Load a configuration from a Json file and returns a configuration.
// See file `settings.json.sample` to see how settings should be formatted.
func (t Bot) LoadSettings(filename string) (Configuration, error) {
	configuration := Configuration{}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("FATAL: Unable to find file "+filename, err)
		return configuration, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("FATAL: Unable to read file "+filename+"! Please copy from settings.json.sample", err)
		return configuration, err
	}
	return configuration, nil
}

// Returns the telegram poll URL, used to retrive messages.
// The URL is built using the loaded configuration.
func (t Bot) telegramPollURL(conf Configuration, offset string) string {
	BASE_URL := "https://api.telegram.org/bot" + conf.ApiKey + "/"
	if len(conf.Timeout) == 0 || conf.Timeout == "0" {
		conf.Timeout = "60"
	}
	if _, err := strconv.Atoi(conf.Timeout); err != nil {
		conf.Timeout = "60"
	}
	POLL_URL := BASE_URL + "getUpdates?offset=" + offset + "&timeout=" + conf.Timeout
	return POLL_URL
}

// Returns the telegram poll URL, used to send messages.
// The URL is built using the loaded configuration.
func (t Bot) telegramSendURL(conf Configuration) string {
	SEND_URL := "https://api.telegram.org/bot" + conf.ApiKey + "/sendMessage"
	return SEND_URL
}

// Retrieve the received telegram messages. Parameter mess is a channel of
// TeleResults messages, and it's used to push received and parsed messages.
// Parameter conf is used to build the URLs.
func (t Bot) getTeleMessages(mess chan teleResults, conf Configuration) {

	var (
		err    error              // Error variable
		req    *http.Request      // HTTP Request
		resp   *http.Response     // HTTP Response
		max_id int64          = 0 // MAX_ID of last received message
	)

	// Creates the HTTP client
	client := &http.Client{}

	for {
		req_url := t.telegramPollURL(conf, strconv.FormatInt(max_id+1, 10))
		// fmt.Println("DEBUG: " + req_url)

		if req, err = http.NewRequest("GET", req_url, nil); err != nil {
			fmt.Printf("FATAL: Could not parse request: %v\n", err)
			os.Exit(1)
		}
		// fmt.Println("DEBUG: Blocked on GET\n")

		if resp, err = client.Do(req); err != nil {
			fmt.Printf("FATAL: Could not send request: %v\n", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("WARN: Malformed body from Telegram!\n")
			continue
		}

		// fmt.Println("DEBUG: Received data from Telegram " + string(body))
		var messages teleAnswer

		if err = json.Unmarshal(body, &messages); err != nil {
			if strings.HasPrefix(string(body), "<!DOCTYPE html>") {
				fmt.Println("FATAL: Wrong API Key, Ask @BotFather! Exiting...")
				os.Exit(1)
			} else {
				fmt.Println("WARN: Telegram JSON Error: " + err.Error())
			}
		} else {
			fmt.Println("INFO: ##### Received messages: " + strconv.Itoa(len(messages.Result)))
			if messages.Ok == false {
				switch messages.Error {
				case 401, 403:
					fmt.Println("FATAL: Wrong API Key, Ask @BotFather! Exiting...")
					os.Exit(1)
				case 400, 404:
					fmt.Println("FATAL: Wrong interaction with Telegram. Please update the telebot library")
					os.Exit(1)
				}
				if messages.Error >= 500 {
					fmt.Println("WARN: Telegram Server error!")
					time.Sleep(30 * time.Second)
					continue
				}
			} else {
				// Messages Ok == true
				for _, msg := range messages.Result {
					if msg.Updid > max_id {
						mess <- msg
						// Update the message ID
						max_id = msg.Updid
					}
				}
			}
		}
	}
}

// Handle the received telegram messages and send answers to the users.
// Parameter infomes is a challel of received Telegram messages. Parameter
// conf is used to build URL and parameter resp is used to create answers.
func (t Bot) handleTeleMessages(infomes chan teleResults, conf Configuration, resp Responder) {

	var err error

	SEND_URL := t.telegramSendURL(conf)
	client := &http.Client{}

	fmt.Println("INFO: Message Handler is ready to answer...")

	for {
		message := <-infomes
		fmt.Println("INFO: Message: '" + message.Message.Text + "' From: '" + message.Message.Chat.Uname + "'")
		// Answer message
		answer := t.getResponse(message.Message.Text, conf, resp)

		vals := url.Values{
			"chat_id": {strconv.FormatInt(message.Message.Chat.Chatid, 10)},
			"text":    {answer}}
		if _, err = client.PostForm(SEND_URL, vals); err != nil {
			fmt.Printf("WARN: Could not send post request: %v\n", err)
			continue
		} else {
			fmt.Println("INFO: Answer: '" + answer + "' To: '" + message.Message.From.Uname + "'")
		}
	}
}

// Process a single user message and returns the answer.
// This method will remove the @BotName (e.g. /start@TestBot) from received message
// to allow a unique interpretation of messages
func (t Bot) getResponse(message string, conf Configuration, resp Responder) string {

	var answer string
	var err error
	message = strings.Replace(message, "@"+conf.BotName, "", 1)

	answer, err = resp(message)
	if err != nil {
		answer = "I'm not able to answer :("
	}
	return answer
}
