// this package manages the messages
package messages

import (
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// TODO: move these funcs/interface to channels
type UiMessageHandler interface {
	NewMessage(Message)
	NewScreen([]Message)
	SetChats([]Chat)
	SetContacts([]Contact)
	SetStatuses([]Message)
	RefreshLanguage()
	PrintError(error)
	PrintText(string)
	SetStatus(SessionStatus)
	OpenFile(string)
	SetQRCode(string)
}

// data struct for current session status
type SessionStatus struct {
	BatteryCharge    int
	BatteryLoading   bool
	BatteryPowersave bool
	Connected        bool
	LastSeen         string
}

// message struct for battery messages
type BatteryMsg struct {
	charge    int
	loading   bool
	powersave bool
}

// message struct for status messages
type StatusMsg struct {
	connected bool
	err       error
}

// message object for commands
type Command struct {
	Name   string
	Params []string
}

type MessageKind string

const (
	MessageKindText     MessageKind = "text"
	MessageKindImage    MessageKind = "image"
	MessageKindVideo    MessageKind = "video"
	MessageKindAudio    MessageKind = "audio"
	MessageKindDocument MessageKind = "document"
	MessageKindUnknown  MessageKind = "unknown"
)

// internal message representation to abstract from message lib
type Message struct {
	Id           string
	ChatId       string // the source of the message (group id or contact id)
	SenderId     string
	ContactId    string
	ContactName  string
	ContactShort string
	Timestamp    uint64
	FromMe       bool
	Forwarded    bool
	IsStatus     bool
	Text         string
	Kind         MessageKind
	MimeType     string
	FileName     string
	Unread       bool
	RawMessage   *waProto.Message
}

// internal contact representation to abstract from message lib
type Chat struct {
	Id      string
	IsGroup bool
	Name    string
	Unread  int
	//TODO: convert to uint64
	LastMessage int64
}

type Contact struct {
	Id    string
	Name  string
	Short string
}

const GROUPSUFFIX = "@g.us"
const CONTACTSUFFIX = "@s.whatsapp.net"
const STATUSSUFFIX = "status@broadcast"

// commandNames lists every supported slash command (without the prefix).
// Keep in sync with the execCommand switch in session_manager.go and the
// special-cased commands (help/commands/quit) handled in main.go.
var commandNames = []string{
	"backlog", "login", "connect", "reset", "disconnect", "logout",
	"send", "select", "read", "info", "download", "open", "show", "url",
	"upload", "sendimage", "sendvideo", "sendaudio", "revoke",
	"leave", "create", "add", "remove", "admin", "removeadmin", "subject",
	"colorlist", "more", "lang",
	"help", "commands", "quit",
}

// AvailableCommands returns the supported slash commands (without prefix).
func AvailableCommands() []string {
	return commandNames
}
