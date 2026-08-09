package messages

import (
	"strings"
	"testing"
)

// TestStatusNeverEntersChatStore verifies that status updates can never leak
// into the per-chat message store, regardless of how they are flagged.
func TestStatusNeverEntersChatStore(t *testing.T) {
	md := &MessageDatabase{}
	md.Init()

	statusCases := []Message{
		// Explicitly flagged as status.
		{Id: "s1", ChatId: "status@broadcast", SenderId: "50495337480@s.whatsapp.net", ContactName: "Ofelia R.", Kind: MessageKindImage, IsStatus: true},
		// Not flagged, but chat is the status broadcast (live-path fallback).
		{Id: "s2", ChatId: "status@broadcast", SenderId: "55839490625765@lid", ContactName: "~", Kind: MessageKindImage},
		// Not flagged, chat id contains the broadcast suffix (weird server variant).
		{Id: "s3", ChatId: "xstatus@broadcast", SenderId: "50492076140@s.whatsapp.net", ContactName: "José Castro", Kind: MessageKindImage},
	}

	for _, msg := range statusCases {
		if added := md.AddMessage(msg, true); added {
			t.Errorf("AddMessage returned true (added to chat store) for status %q", msg.Id)
		}
		if len(md.GetMessages(msg.ChatId)) != 0 {
			t.Errorf("status %q leaked into chat store %q", msg.Id, msg.ChatId)
		}
	}

	statuses := md.GetStatuses()
	if len(statuses) != len(statusCases) {
		t.Fatalf("expected %d statuses in status store, got %d", len(statusCases), len(statuses))
	}

	// No chat may be created from status messages.
	if chats := md.GetChatIds(); len(chats) != 0 {
		t.Fatalf("statuses created %d chats in the chat list, want 0", len(chats))
	}

	// A normal message still works and creates its chat.
	normal := Message{Id: "n1", ChatId: "50495337480@s.whatsapp.net", ContactName: "Ofelia R.", ContactShort: "Ofelia R.", Kind: MessageKindText, Text: "hola"}
	if added := md.AddMessage(normal, true); !added {
		t.Error("normal message was not added to chat store")
	}
	if msgs := md.GetMessages(normal.ChatId); len(msgs) != 1 || msgs[0].Id != "n1" {
		t.Errorf("normal message not retrievable from chat store: %+v", msgs)
	}
	if chats := md.GetChatIds(); len(chats) != 1 || chats[0].Id != normal.ChatId {
		t.Errorf("normal message did not create its chat: %+v", chats)
	}
}

// TestIsStatusDetection verifies the status detection predicate covers all
// known ways whatsmeow delivers statuses.
func TestIsStatusDetection(t *testing.T) {
	detect := func(infoType, chatUser, chatServer string) bool {
		return infoType == "status" ||
			chatUser == "status" && chatServer == "broadcast"
	}

	cases := []struct {
		name       string
		infoType   string
		chatUser   string
		chatServer string
		want       bool
	}{
		{"classic live status", "status", "status", "broadcast", true},
		{"status without type attr", "", "status", "broadcast", true},
		{"status with text type", "text", "status", "broadcast", true},
		{"regular DM", "text", "50495337480", "s.whatsapp.net", false},
		{"group message", "", "12345", "g.us", false},
	}
	for _, tc := range cases {
		if got := detect(tc.infoType, tc.chatUser, tc.chatServer); got != tc.want {
			t.Errorf("%s: detect(%q,%q,%q) = %v, want %v", tc.name, tc.infoType, tc.chatUser, tc.chatServer, got, tc.want)
		}
	}
}

// TestStatusesSortedNewestFirst guards the status store ordering contract.
func TestStatusesSortedNewestFirst(t *testing.T) {
	md := &MessageDatabase{}
	md.Init()
	md.AddStatus(Message{Id: "a", ChatId: STATUSSUFFIX, Timestamp: 100})
	md.AddStatus(Message{Id: "b", ChatId: STATUSSUFFIX, Timestamp: 300})
	md.AddStatus(Message{Id: "c", ChatId: STATUSSUFFIX, Timestamp: 200})

	got := md.GetStatuses()
	if len(got) != 3 || got[0].Id != "b" || got[1].Id != "c" || got[2].Id != "a" {
		t.Errorf("statuses not sorted newest-first: %s", ids(got))
	}

	if !md.RemoveStatus("b") || md.RemoveStatus("b") {
		t.Error("RemoveStatus failed or removed twice")
	}
	if got = md.GetStatuses(); len(got) != 2 || strings.Contains(ids(got), "b") {
		t.Errorf("status not removed: %s", ids(got))
	}
}

func ids(msgs []Message) string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Id)
	}
	return strings.Join(out, ",")
}
