// Package for creating a running a simple Telegram bot.
// This bot is capable just to answer simple user/group messages,
// all the logic must be implemented inside a Responder func
package telebot

// A general answer from Telegram API
type teleAnswer struct {
	Ok     bool          `json:"ok"`
	Result []teleResults `json:"result"`
	Error  int           `json:"error_code"`
}

// A telegram resource: id + message
type teleResults struct {
	Updid   int64       `json:"update_id"`
	Message teleMessage `json:"message"`
}

// Details about the specific message
type teleMessage struct {
	Text  string   `json:"text"`
	Mesid int64    `json:"message_id"`
	From  teleFrom `json:"from"`
	Chat  teleChat `json:"chat"`
	Date  int64    `json:"date"`
}

// Details about the sender of the message
type teleFrom struct {
	Frmid   int64  `json:"id"`
	Fstname string `json:"first_name"`
	Sndname string `json:"last_name"`
	Uname   string `json:"username"`
}

// Details about the chat of the message
type teleChat struct {
	Chatid  int64  `json:"id"`
	Fstname string `json:"first_name"`
	Sndname string `json:"last_name"`
	Uname   string `json:"username"`
}
