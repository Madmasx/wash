package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
	"github.com/zyedidia/clipboard"
	"wash/config"
	"wash/messages"
)

var VERSION string = "v2.2.0"

var sndTxt string = ""
var currentReceiver messages.Chat = messages.Chat{}
var curRegions []messages.Message

var textView *tview.TextView
var treeView *tview.TreeView
var textInput *tview.InputField
var topBar *tview.TextView
var infoBar *tview.TextView

var chatRoot *tview.TreeNode
var groupRoot *tview.TreeNode
var statusRoot *tview.TreeNode
var contactRoot *tview.TreeNode
var app *tview.Application
var pages *tview.Pages

// contactList keeps the loaded contacts for @mention autocomplete and for
// resolving @Name into a direct conversation. It is refreshed by SetContacts.
var contactList []messages.Contact

var sessionManager *messages.SessionManager

var keyBindings *cbind.Configuration

var uiHandler messages.UiMessageHandler

// lastQRText keeps the rendered QR block currently shown in the chat, so
// SetQRCode can replace it in place on every refresh instead of stacking
// copies. It is only touched from the UI (QueueUpdateDraw) goroutine.
var lastQRText string

// lastHelpOverlay keeps the exact text of the last help/commands overlay
// printed into the view, so renderHelpOverlay can replace it in place instead
// of stacking duplicates every time /help, /commands or Ctrl+p is pressed.
var lastHelpOverlay string

// mediaIndexByChat maps, per chat, the ordered ids of the media messages shown
// in that chat's view, so /show N can resolve a per-chat image number (1..K,
// reset per chat) to its message id. It is only touched from the UI goroutine.
var mediaIndexByChat map[string][]string

func main() {
	config.InitConfig()
	config.MigrateLegacySession()
	uiHandler = UiHandler{}
	sessionManager = &messages.SessionManager{}
	sessionManager.Init(uiHandler)

	app = tview.NewApplication()
	mediaIndexByChat = make(map[string][]string)

	sideBarWidth := config.Config.Ui.ChatSidebarWidth
	gridLayout := tview.NewGrid()
	gridLayout.SetRows(1, 0, 1)
	gridLayout.SetColumns(sideBarWidth, 0, sideBarWidth)
	gridLayout.SetBorders(true)
	gridLayout.SetBackgroundColor(tcell.ColorNames[config.Config.Colors.Background])
	gridLayout.SetBordersColor(tcell.ColorNames[config.Config.Colors.Borders])

	cmdPrefix := config.Config.General.CmdPrefix
	topBar = tview.NewTextView()
	topBar.SetDynamicColors(true)
	topBar.SetScrollable(false)
	topBar.SetText("[::b] WaSh " + VERSION + "  [-::d]Type " + cmdPrefix + "help or press " + config.Config.Keymap.CommandHelp + " for help")
	topBar.SetBackgroundColor(tcell.ColorNames[config.Config.Colors.Background])

	infoBar = tview.NewTextView()
	infoBar.SetDynamicColors(true)
	infoBar.SetBackgroundColor(tcell.ColorNames[config.Config.Colors.Background])
	UpdateStatusBar(messages.SessionStatus{})

	textView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	textView.SetBackgroundColor(tcell.ColorNames[config.Config.Colors.Background])
	textView.SetTextColor(tcell.ColorNames[config.Config.Colors.Text])
	textView.SetMouseCapture(linkClickCapture)

	renderHelpOverlay(PrintHelp)

	textInput = tview.NewInputField()
	textInput.SetBackgroundColor(tcell.ColorNames[config.Config.Colors.Background])
	textInput.SetFieldBackgroundColor(tcell.ColorNames[config.Config.Colors.InputBackground])
	textInput.SetFieldTextColor(tcell.ColorNames[config.Config.Colors.InputText])
	textInput.SetChangedFunc(func(change string) {
		sndTxt = change
	})
	textInput.SetDoneFunc(EnterCommand)
	textInput.SetAutocompleteFunc(inputSuggestions)
	textInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp {
			// While the autocomplete list (commands or @mentions) is open,
			// the arrows navigate the suggestions instead of scrolling the
			// history.
			if len(inputSuggestions(sndTxt)) > 0 {
				return event
			}
			offset, _ := textView.GetScrollOffset()
			if event.Key() == tcell.KeyDown {
				offset += 1
			} else {
				offset -= 1
			}
			textView.ScrollTo(offset, 0)
			return nil
		}
		if event.Key() == tcell.KeyPgDn {
			offset, _ := textView.GetScrollOffset()
			offset += 10
			textView.ScrollTo(offset, 0)
			return nil
		}
		if event.Key() == tcell.KeyPgUp {
			offset, _ := textView.GetScrollOffset()
			offset -= 10
			textView.ScrollTo(offset, 0)
			return nil
		}
		return event
	})

	gridLayout.AddItem(topBar, 0, 0, 1, 4, 0, 0, false)
	gridLayout.AddItem(infoBar, 2, 0, 1, 1, 0, 0, false)
	gridLayout.AddItem(MakeTree(), 1, 0, 1, 1, 0, 0, false)
	gridLayout.AddItem(textView, 1, 1, 1, 3, 0, 0, false)
	gridLayout.AddItem(textInput, 2, 1, 1, 3, 0, 0, false)

	pages = tview.NewPages()
	pages.AddPage("main", gridLayout, true, true)

	app.SetRoot(pages, true)
	app.EnableMouse(true)
	app.SetFocus(textInput)
	if err := sessionManager.StartManager(); err != nil {
		PrintError(err)
	}
	LoadShortcuts()
	app.Run()
}

// creates the TreeView for chats, statuses and contacts
func MakeTree() *tview.TreeView {
	mainRoot := tview.NewTreeNode("WaSh").
		SetColor(tcell.ColorNames[config.Config.Colors.ListHeader])

	chatRoot = tview.NewTreeNode(config.T("ui.chats")).
		SetColor(tcell.ColorNames[config.Config.Colors.ListHeader])
	groupRoot = tview.NewTreeNode(config.T("ui.groups")).
		SetColor(tcell.ColorNames[config.Config.Colors.ListHeader])
	statusRoot = tview.NewTreeNode(config.T("ui.statuses")).
		SetColor(tcell.ColorNames[config.Config.Colors.ListHeader])
	contactRoot = tview.NewTreeNode(config.T("ui.contacts")).
		SetColor(tcell.ColorNames[config.Config.Colors.ListHeader])

	mainRoot.AddChild(chatRoot)
	mainRoot.AddChild(groupRoot)
	mainRoot.AddChild(statusRoot)
	mainRoot.AddChild(contactRoot)

	treeView = tview.NewTreeView().
		SetRoot(mainRoot).
		SetCurrentNode(chatRoot)
	treeView.SetBackgroundColor(tcell.ColorNames[config.Config.Colors.Background])

	// If a chat, status or contact was selected, open it on Enter, or toggle expansion.
	treeView.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		children := node.GetChildren()
		if reference == nil || len(children) > 0 {
			node.SetExpanded(!node.IsExpanded())
			return
		}
		switch ref := reference.(type) {
		case messages.Chat:
			SetDisplayedChat(ref)
		case messages.Contact:
			chat := messages.Chat{
				Id:      ref.Id,
				IsGroup: false,
				Name:    ref.Name,
			}
			SetDisplayedChat(chat)
		case messages.Message:
			if ref.IsStatus || ref.ChatId == messages.STATUSSUFFIX {
				// Statuses live in the statuses store, not in the per-chat
				// message maps: show the status itself instead of an empty chat.
				currentReceiver = messages.Chat{Id: ref.ChatId, Name: ref.ContactName}
				textView.Clear()
				textView.SetTitle(ref.ContactName)
				uiHandler.NewScreen([]messages.Message{ref})
				return
			}
			chat := messages.Chat{
				Id:      ref.ChatId,
				IsGroup: false,
				Name:    ref.ContactName,
			}
			SetDisplayedChat(chat)
		}
	})
	return treeView
}

func handleFocusMessage(ev *tcell.EventKey) *tcell.EventKey {
	if !textView.HasFocus() {
		app.SetFocus(textView)
		if curRegions != nil && len(curRegions) > 0 {
			textView.Highlight(curRegions[len(curRegions)-1].Id)
		}
	}
	return nil
}

func handleFocusInput(ev *tcell.EventKey) *tcell.EventKey {
	ResetMsgSelection()
	if !textInput.HasFocus() {
		app.SetFocus(textInput)
	}
	return nil
}

func handleFocusContacts(ev *tcell.EventKey) *tcell.EventKey {
	ResetMsgSelection()
	if !treeView.HasFocus() {
		app.SetFocus(treeView)
	}
	return nil
}

func handleSwitchPanels(ev *tcell.EventKey) *tcell.EventKey {
	ResetMsgSelection()
	if !textInput.HasFocus() {
		app.SetFocus(textInput)
	} else {
		app.SetFocus(treeView)
	}
	return nil
}

func handleCommand(command string) func(ev *tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		sessionManager.CommandChannel <- messages.Command{command, nil}
		return nil
	}
}

func handleCopyUser(ev *tcell.EventKey) *tcell.EventKey {
	if hls := textView.GetHighlights(); len(hls) > 0 {
		for _, val := range curRegions {
			if val.Id == hls[0] {
				clipboard.WriteAll(val.ContactId, "clipboard")
				PrintText("copied id of " + val.ContactName + " to clipboard")
			}
		}
		ResetMsgSelection()
	} else if currentReceiver.Id != "" {
		clipboard.WriteAll(currentReceiver.Id, "clipboard")
		PrintText("copied id of " + currentReceiver.Name + " to clipboard")
	}
	return nil
}

func handlePasteUser(ev *tcell.EventKey) *tcell.EventKey {
	// If the clipboard holds an image (e.g. a fresh screenshot), send it as
	// an attachment instead of pasting text.
	if path, err := clipboardImagePath(); err == nil {
		if currentReceiver.Id == "" {
			PrintText(config.T("ui.no_receiver"))
			return nil
		}
		sessionManager.CommandChannel <- messages.Command{"sendimage", []string{path}}
		PrintText(config.T("ui.paste_image_sent") + " " + currentReceiver.Name)
		return nil
	}
	// No image in the clipboard: fall back to the plain text paste.
	if clip, err := safeReadClipboard(); err == nil {
		textInput.SetText(textInput.GetText() + " " + clip)
	} else {
		PrintError(err)
	}
	return nil
}

// clipboardImagePath writes the clipboard image (if any) to a temporary file
// and returns its path. It returns an error when the clipboard holds no image
// or no clipboard tool is available.
func clipboardImagePath() (string, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		wlPaste, err := exec.LookPath("wl-paste")
		if err != nil {
			return "", errors.New("wl-paste not found (install wl-clipboard)")
		}
		out, err := exec.Command(wlPaste, "--list-types").Output()
		if err != nil {
			return "", fmt.Errorf("wl-paste: %v", err)
		}
		mime := pickClipboardImageType(strings.Split(string(out), "\n"))
		if mime == "" {
			return "", errors.New("no image in clipboard")
		}
		data, err := exec.Command(wlPaste, "--type", mime).Output()
		if err != nil {
			return "", fmt.Errorf("wl-paste: %v", err)
		}
		return writeTempImage(data, strings.TrimPrefix(mime, "image/"))
	}

	if os.Getenv("DISPLAY") != "" {
		xclip, err := exec.LookPath("xclip")
		if err != nil {
			return "", errors.New("xclip not found (install xclip)")
		}
		out, err := exec.Command(xclip, "-selection", "clipboard", "-t", "TARGETS", "-o").Output()
		if err != nil {
			return "", fmt.Errorf("xclip: %v", err)
		}
		mime := pickClipboardImageType(strings.Split(string(out), "\n"))
		if mime == "" {
			return "", errors.New("no image in clipboard")
		}
		data, err := exec.Command(xclip, "-selection", "clipboard", "-t", mime, "-o").Output()
		if err != nil {
			return "", fmt.Errorf("xclip: %v", err)
		}
		return writeTempImage(data, strings.TrimPrefix(mime, "image/"))
	}
	return "", errors.New("no display available")
}

// pickClipboardImageType returns the first image/* mime type from the given
// clipboard target list, or "" if there is none.
func pickClipboardImageType(targets []string) string {
	for _, t := range targets {
		t = strings.TrimSpace(t)
		if strings.HasPrefix(t, "image/") {
			return t
		}
	}
	return ""
}

// writeTempImage writes image data to a temp file with the proper extension.
func writeTempImage(data []byte, ext string) (string, error) {
	if len(data) == 0 {
		return "", errors.New("empty clipboard image data")
	}
	if ext == "jpeg" {
		ext = "jpg"
	}
	f, err := os.CreateTemp("", "wash-paste-*."+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func safeReadClipboard() (clip string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("clipboard paste is unavailable: %v", rec)
		}
	}()
	return clipboard.ReadAll("clipboard")
}

func handleQuit(ev *tcell.EventKey) *tcell.EventKey {
	sessionManager.CommandChannel <- messages.Command{"disconnect", nil}
	app.Stop()
	return nil
}

func handleHelp(ev *tcell.EventKey) *tcell.EventKey {
	renderHelpOverlay(PrintHelp, PrintCommands)
	return nil
}

func handleMessageCommand(command string) func(ev *tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		hls := textView.GetHighlights()
		if len(hls) > 0 {
			sessionManager.CommandChannel <- messages.Command{command, []string{hls[0]}}
			ResetMsgSelection()
			app.SetFocus(textInput)
		}
		return nil
	}
}

func handleMessagesMove(amount int) func(ev *tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		if curRegions == nil || len(curRegions) == 0 {
			return nil
		}
		hls := textView.GetHighlights()
		if len(hls) > 0 {
			newId := GetOffsetMsgId(hls[0], amount)
			if newId != "" {
				textView.Highlight(newId)
			}
		} else {
			if amount < 0 {
				textView.Highlight(curRegions[0].Id)
			} else {
				textView.Highlight(curRegions[len(curRegions)-1].Id)
			}
		}
		textView.ScrollToHighlight()
		return nil
	}
}

func handleChatPanelUp(ev *tcell.EventKey) *tcell.EventKey {
	//TODO: scroll selection in treeView? or chatRoot? How?
	return ev
}

func handleChatPanelDown(ev *tcell.EventKey) *tcell.EventKey {
	return ev
}

func handleMessagesLast(ev *tcell.EventKey) *tcell.EventKey {
	if curRegions == nil || len(curRegions) == 0 {
		return nil
	}
	textView.Highlight(curRegions[len(curRegions)-1].Id)
	textView.ScrollToHighlight()
	return nil
}

func handleMessagesFirst(ev *tcell.EventKey) *tcell.EventKey {
	if curRegions == nil || len(curRegions) == 0 {
		return nil
	}
	textView.Highlight(curRegions[0].Id)
	textView.ScrollToHighlight()
	return nil
}

func handleExitMessages(ev *tcell.EventKey) *tcell.EventKey {
	if curRegions == nil || len(curRegions) == 0 {
		return nil
	}
	ResetMsgSelection()
	app.SetFocus(textInput)
	return nil
}

// load the key map
func LoadShortcuts() {
	// global bindings for app
	keyBindings = cbind.NewConfiguration()
	if err := keyBindings.Set(config.Config.Keymap.FocusMessages, handleFocusMessage); err != nil {
		PrintErrorMsg("focus_messages:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.FocusInput, handleFocusInput); err != nil {
		PrintErrorMsg("focus_input:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.FocusChats, handleFocusContacts); err != nil {
		PrintErrorMsg("focus_contacts:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.SwitchPanels, handleSwitchPanels); err != nil {
		PrintErrorMsg("switch_panels:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.CommandRead, handleCommand("read")); err != nil {
		PrintErrorMsg("command_read:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.Copyuser, handleCopyUser); err != nil {
		PrintErrorMsg("copyuser:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.Pasteuser, handlePasteUser); err != nil {
		PrintErrorMsg("pasteuser:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.CommandBacklog, handleCommand("backlog")); err != nil {
		PrintErrorMsg("command_backlog:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.CommandConnect, handleCommand("login")); err != nil {
		PrintErrorMsg("command_connect:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.CommandQuit, handleQuit); err != nil {
		PrintErrorMsg("command_quit:", err)
	}
	if err := keyBindings.Set(config.Config.Keymap.CommandHelp, handleHelp); err != nil {
		PrintErrorMsg("command_help:", err)
	}
	app.SetInputCapture(keyBindings.Capture)
	// bindings for chat message text view
	keysMessages := cbind.NewConfiguration()
	if err := keysMessages.Set(config.Config.Keymap.MessageDownload, handleMessageCommand("download")); err != nil {
		PrintErrorMsg("message_download:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.MessageOpen, handleMessageCommand("open")); err != nil {
		PrintErrorMsg("message_open:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.Copyuser, handleCopyUser); err != nil {
		PrintErrorMsg("copyuser:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.Pasteuser, handlePasteUser); err != nil {
		PrintErrorMsg("pasteuser:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.MessageShow, handleMessageCommand("show")); err != nil {
		PrintErrorMsg("message_show:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.MessageUrl, handleMessageCommand("url")); err != nil {
		PrintErrorMsg("message_url:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.MessageInfo, handleMessageCommand("info")); err != nil {
		PrintErrorMsg("message_info:", err)
	}
	if err := keysMessages.Set(config.Config.Keymap.MessageRevoke, handleMessageCommand("revoke")); err != nil {
		PrintErrorMsg("message_revoke:", err)
	}
	keysMessages.SetKey(tcell.ModNone, tcell.KeyEscape, handleExitMessages)
	keysMessages.SetKey(tcell.ModNone, tcell.KeyUp, handleMessagesMove(-1))
	keysMessages.SetKey(tcell.ModNone, tcell.KeyDown, handleMessagesMove(1))
	keysMessages.SetKey(tcell.ModNone, tcell.KeyPgUp, handleMessagesMove(-10))
	keysMessages.SetKey(tcell.ModNone, tcell.KeyPgDn, handleMessagesMove(10))
	keysMessages.SetRune(tcell.ModNone, 'k', handleMessagesMove(-1))
	keysMessages.SetRune(tcell.ModNone, 'j', handleMessagesMove(1))
	keysMessages.SetRune(tcell.ModNone, 'g', handleMessagesFirst)
	keysMessages.SetRune(tcell.ModNone, 'G', handleMessagesLast)
	keysMessages.SetRune(tcell.ModCtrl, 'u', handleMessagesMove(-10))
	keysMessages.SetRune(tcell.ModCtrl, 'd', handleMessagesMove(10))
	textView.SetInputCapture(keysMessages.Capture)
	keysChatPanel := cbind.NewConfiguration()
	keysChatPanel.SetRune(tcell.ModCtrl, 'u', handleChatPanelUp)
	keysChatPanel.SetRune(tcell.ModCtrl, 'd', handleChatPanelDown)
	treeView.SetInputCapture(keysChatPanel.Capture)
}

// prints help to chat view
// renderHelpOverlay prints the given help/commands sections, replacing in
// place whatever overlay was rendered last. The rest of the view — chat
// messages or the pairing QR block — is never touched, so calling it several
// times (boot, /help, Ctrl+p, /commands) does not stack copies.
func renderHelpOverlay(parts ...func()) {
	body := textView.GetText(false)
	if lastHelpOverlay != "" {
		if idx := strings.LastIndex(body, lastHelpOverlay); idx >= 0 {
			body = body[:idx] + body[idx+len(lastHelpOverlay):]
		}
	}
	textView.SetText(body)
	startLen := len(body)
	for _, p := range parts {
		p()
	}
	lastHelpOverlay = textView.GetText(false)[startLen:]
}

func PrintHelp() {
	cmdPrefix := config.Config.General.CmdPrefix
	tviewLine("[-::u]" + config.T("help.keys") + "[-::-]")
	tviewLine("")
	tviewLine(config.T("help.global"))
	tviewLine("[::b] Up/Down[::-] = " + config.T("help.scroll"))
	tviewLine("[::b]", config.Config.Keymap.SwitchPanels, "[::-] = "+config.T("help.switch_input"))
	tviewLine("[::b]", config.Config.Keymap.FocusMessages, "[::-] = "+config.T("help.focus_msg"))
	tviewLine("[::b]", config.Config.Keymap.CommandQuit, "[::-] = "+config.T("help.exit"))
	tviewLine("[::b] " + cmdPrefix + "lang[::-] [es|en] = " + config.T("help.lang"))
	tviewLine("[::b] Drop[::-] file = " + config.T("help.dragdrop"))
	tviewLine("[::b] @Name[::-] = " + config.T("help.mention"))
	tviewLine("")
	tviewLine("[-::-]" + config.T("help.msg_panel") + "[-::-]")
	tviewLine("[::b] Up/Down[::-] = " + config.T("help.select_msg"))
	tviewLine("[::b]", config.Config.Keymap.MessageDownload, "[::-] = "+config.T("help.download"))
	tviewLine("[::b]", config.Config.Keymap.MessageOpen, "[::-] = "+config.T("help.open"))
	tviewLine("[::b]", config.Config.Keymap.MessageShow, "[::-] = "+config.T("help.show"))
	tviewLine("[::d] " + config.T("help.show_index") + "[::-]")
	tviewLine("[::b]", config.Config.Keymap.MessageUrl, "[::-] = "+config.T("help.url"))
	tviewLine("[::d] " + config.T("help.click_link"))
	tviewLine("[::b]", config.Config.Keymap.MessageRevoke, "[::-] = "+config.T("help.revoke"))
	tviewLine("[::b]", config.Config.Keymap.MessageInfo, "[::-] = "+config.T("help.info"))
	tviewLine("")
	tviewLine(config.T("help.config_file"), config.AbbreviateHome(config.GetConfigFilePath()))
	tviewLine("")
	tviewLine(fmt.Sprintf(config.T("help.type_commands"), cmdPrefix+"commands"))
	tviewLine("")
}

func PrintCommands() {
	cmdPrefix := config.Config.General.CmdPrefix
	tviewLine("")
	tviewLine("[-::u]" + config.T("cmds.commands") + "[-::-]")
	tviewLine("")
	tviewLine("[-::-]" + config.T("cmds.global") + "[-::-]")
	tviewLine("[::b] "+cmdPrefix+"connect [::-]or[::b]", config.Config.Keymap.CommandConnect, "[::-] = "+config.T("cmds.connect"))
	tviewLine("[::b] " + cmdPrefix + "disconnect[::-]  = " + config.T("cmds.disconnect"))
	tviewLine("[::b] " + cmdPrefix + "logout[::-]  = " + config.T("cmds.logout"))
	tviewLine("[::b] " + cmdPrefix + "reset[::-]  = " + config.T("cmds.reset"))
	tviewLine("[::b] "+cmdPrefix+"quit [::-]or[::b]", config.Config.Keymap.CommandQuit, "[::-] = "+config.T("cmds.quit"))
	tviewLine("[::b] " + cmdPrefix + "lang[::-] [es|en]  = " + config.T("cmds.lang"))
	tviewLine("")
	tviewLine("[-::-]" + config.T("cmds.chat") + "[-::-]")
	tviewLine("[::b] "+cmdPrefix+"backlog [::-]or[::b]", config.Config.Keymap.CommandBacklog, "[::-] = "+fmt.Sprintf(config.T("cmds.backlog"), config.Config.General.BacklogMsgQuantity))
	tviewLine("[::b] "+cmdPrefix+"read [::-]or[::b]", config.Config.Keymap.CommandRead, "[::-] = "+config.T("cmds.read"))
	tviewLine("[::b] " + cmdPrefix + "upload[::-] /path/to/file  = " + config.T("cmds.upload"))
	tviewLine("[::b] " + cmdPrefix + "sendimage[::-] /path/to/file  = " + config.T("cmds.sendimage"))
	tviewLine("[::b] " + cmdPrefix + "sendvideo[::-] /path/to/file  = " + config.T("cmds.sendvideo"))
	tviewLine("[::b] " + cmdPrefix + "sendaudio[::-] /path/to/file  = " + config.T("cmds.sendaudio"))
	tviewLine("[::b] " + cmdPrefix + "show[::-] [N|message-id[]  = " + config.T("cmds.show"))
	tviewLine("")
	tviewLine("[-::-]" + config.T("cmds.groups") + "[-::-]")
	tviewLine("[::b] " + cmdPrefix + "leave[::-]  = " + config.T("cmds.leave"))
	tviewLine("[::b] " + cmdPrefix + "create[::-] [user-id[] [user-id[] Group Subject  = " + config.T("cmds.create"))
	tviewLine("[::b] " + cmdPrefix + "subject[::-] New Subject  = " + config.T("cmds.subject"))
	tviewLine("[::b] " + cmdPrefix + "add[::-] [user-id[]  = " + config.T("cmds.add"))
	tviewLine("[::b] " + cmdPrefix + "remove[::-] [user-id[]  = " + config.T("cmds.remove"))
	tviewLine("[::b] " + cmdPrefix + "admin[::-] [user-id[]  = " + config.T("cmds.admin"))
	tviewLine("[::b] " + cmdPrefix + "removeadmin[::-] [user-id[]  = " + config.T("cmds.removeadmin"))
	tviewLine("")
	tviewLine("Use[::b]", config.Config.Keymap.Copyuser, "[::-]"+config.T("cmds.copyid"))
	tviewLine("Use[::b]", config.Config.Keymap.Pasteuser, "[::-]"+config.T("cmds.paste"))
	tviewLine("")
}

// called when text is entered by the user
func EnterCommand(key tcell.Key) {
	if sndTxt == "" {
		return
	}
	if key == tcell.KeyEsc {
		textInput.SetText("")
		return
	}
	cmdPrefix := config.Config.General.CmdPrefix
	if sndTxt == cmdPrefix+"help" {
		renderHelpOverlay(PrintHelp)
		textInput.SetText("")
		return
	}
	if sndTxt == cmdPrefix+"commands" {
		renderHelpOverlay(PrintCommands)
		textInput.SetText("")
		return
	}
	if sndTxt == cmdPrefix+"quit" {
		sessionManager.CommandChannel <- messages.Command{"disconnect", nil}
		app.Stop()
		return
	}
	// Drag & drop: terminal emulators (WezTerm, kitty, …) insert the dropped
	// file's path into the input. Detect it and send it as an attachment.
	if cmdName, path, ok := mediaCommandForPath(sndTxt); ok {
		if currentReceiver.Id == "" {
			PrintText(config.T("ui.no_receiver"))
			textInput.SetText("")
			return
		}
		sessionManager.CommandChannel <- messages.Command{cmdName, []string{path}}
		textInput.SetText("")
		return
	}
	// @mention: open a direct conversation with the named contact.
	// The name may contain spaces (e.g. after selecting it from the
	// autocomplete list with the arrow keys).
	if strings.HasPrefix(sndTxt, "@") {
		if chat, ok := chatForMention(sndTxt[1:]); ok {
			SetDisplayedChat(chat)
		} else {
			PrintText(config.T("ui.contact_not_found") + " " + sndTxt)
		}
		textInput.SetText("")
		return
	}
	if strings.HasPrefix(sndTxt, cmdPrefix) {
		cmd := strings.TrimPrefix(sndTxt, cmdPrefix)
		var params []string
		if strings.Index(cmd, " ") >= 0 {
			cmdParts := strings.Split(cmd, " ")
			cmd = cmdParts[0]
			params = cmdParts[1:]
		}
		// /show N: resolve the per-chat attachment number to its message id, so
		// attachments get opened by the number printed next to them ([#N])
		// without highlighting.
		if cmd == "show" && len(params) == 1 {
			if n, err := strconv.Atoi(params[0]); err == nil {
				if id, ok := resolveMediaNumber(currentReceiver.Id, n); ok {
					params[0] = id
				} else {
					PrintError(fmt.Errorf(config.T("show.no_index"), n))
					return
				}
			}
		}
		sessionManager.CommandChannel <- messages.Command{cmd, params}
		textInput.SetText("")
		return
	}
	if currentReceiver.Id == "" {
		PrintText(config.T("ui.no_receiver"))
		textInput.SetText("")
		return
	}
	// no command, send as message
	msg := messages.Command{
		Name:   "send",
		Params: []string{currentReceiver.Id, expandMentions(sndTxt)},
	}
	sessionManager.CommandChannel <- msg
	textInput.SetText("")
}

// mediaCommandForPath returns the media-send command name and the cleaned
// path when text looks like a local file path (e.g. dropped onto the
// terminal window), so it can be sent as an attachment instead of text.
// The kind is chosen by file extension; anything unknown becomes a document.
func mediaCommandForPath(text string) (string, string, bool) {
	path := strings.TrimSpace(text)
	path = strings.Trim(path, `"'`)
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".avif", ".heic":
		return "sendimage", path, true
	case ".mp4", ".avi", ".mkv", ".mov", ".webm", ".m4v", ".mpg", ".mpeg", ".wmv", ".flv":
		return "sendvideo", path, true
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".opus", ".aac", ".wma":
		return "sendaudio", path, true
	default:
		return "upload", path, true
	}
}

// inputSuggestions is the autocomplete entry point: slash commands when the
// input starts with the command prefix, @contact mentions otherwise.
func inputSuggestions(currentText string) []string {
	if strings.HasPrefix(currentText, config.Config.General.CmdPrefix) {
		return commandSuggestions(currentText)
	}
	return contactSuggestions(currentText)
}

// commandSuggestions returns slash-command completions for the text after
// the command prefix. The full command list is offered when only the prefix
// has been typed; once arguments follow (a space), no suggestions are shown.
func commandSuggestions(currentText string) []string {
	cmdPrefix := config.Config.General.CmdPrefix
	term := strings.TrimPrefix(currentText, cmdPrefix)
	if strings.ContainsAny(term, " \t") {
		return nil
	}
	lower := strings.ToLower(term)
	var suggestions []string
	for _, cmd := range messages.AvailableCommands() {
		if strings.HasPrefix(strings.ToLower(cmd), lower) {
			suggestions = append(suggestions, cmdPrefix+cmd)
		}
	}
	return suggestions
}

// contactSuggestions returns @mention completions based on the contact list.
// The returned entries are the full resulting text (e.g. "hola @Juan"), so
// tview's replacement keeps any text typed before the mention. Suggestions
// are offered while the text after the last "@" still matches a contact —
// including multi-word names — so the drop-down stays open while the user
// navigates it with the arrow keys.
func contactSuggestions(currentText string) []string {
	at := strings.LastIndex(currentText, "@")
	if at < 0 {
		return nil
	}
	term := currentText[at+1:]
	if term == "" {
		return nil
	}
	lower := strings.ToLower(term)
	var suggestions []string
	for _, c := range contactList {
		name := c.Name
		if name == "" {
			name = c.Short
		}
		if name == "" || !strings.Contains(strings.ToLower(name), lower) {
			continue
		}
		suggestions = append(suggestions, currentText[:at+1]+name)
		if len(suggestions) >= 8 {
			break
		}
	}
	return suggestions
}

// chatForMention resolves a contact name typed after "@" to its Chat (JID),
// so @Name can open a direct conversation.
func chatForMention(name string) (messages.Chat, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, c := range contactList {
		candidate := c.Name
		if candidate == "" {
			candidate = c.Short
		}
		if strings.ToLower(strings.TrimSpace(candidate)) == lower {
			return messages.Chat{Id: c.Id, IsGroup: false, Name: candidate}, true
		}
	}
	return messages.Chat{}, false
}

// expandMentions replaces "@Name" mentions in outgoing text with "@<phone>",
// the format WhatsApp uses for real mentions in group chats. Names may
// contain spaces; the regex is built from the contact list (escaped) and
// matched case-insensitively in a single pass.
func expandMentions(text string) string {
	if len(contactList) == 0 {
		return text
	}
	var parts []string
	phones := make(map[string]string) // lowercase name -> @phone
	for _, c := range contactList {
		name := c.Name
		if name == "" {
			name = c.Short
		}
		if name == "" {
			continue
		}
		phone := strings.Split(c.Id, "@")[0]
		if phone == "" {
			continue
		}
		parts = append(parts, regexp.QuoteMeta(name))
		phones[strings.ToLower(name)] = "@" + phone
	}
	if len(parts) == 0 {
		return text
	}
	re := regexp.MustCompile(`(?i)@(` + strings.Join(parts, "|") + `)`)
	return re.ReplaceAllStringFunc(text, func(m string) string {
		if phone, ok := phones[strings.ToLower(m[1:])]; ok {
			return phone
		}
		return m
	})
}

// get the next message id to select (highlighted + offset)
func GetOffsetMsgId(curId string, offset int) string {
	if curRegions == nil || len(curRegions) == 0 {
		return ""
	}
	for idx, val := range curRegions {
		if val.Id == curId {
			arrPos := idx + offset
			if len(curRegions) > arrPos && arrPos >= 0 {
				return curRegions[arrPos].Id
			}
		}
	}
	if offset > 0 {
		return curRegions[0].Id
	} else {
		return curRegions[len(curRegions)-1].Id
	}
}

// resets the selection in the textView and scrolls it down
func ResetMsgSelection() {
	if len(textView.GetHighlights()) > 0 {
		textView.Highlight("")
	}
	textView.ScrollToEnd()
}

// prints text to the TextView
func PrintText(txt string) {
	tviewLine(txt)
}

// prints an error to the TextView
func PrintError(err error) {
	if err == nil {
		return
	}
	tviewLine("["+config.Config.Colors.Negative+"]", err.Error(), "[-]")
}

// prints an error to the TextView
func PrintErrorMsg(text string, err error) {
	if err == nil {
		return
	}
	tviewLine("["+config.Config.Colors.Negative+"]", text, err.Error(), "[-]")
}

// updates the status bar
func UpdateStatusBar(statusInfo messages.SessionStatus) {
	out := " "
	if statusInfo.Connected {
		out += "[" + config.Config.Colors.Positive + "]" + config.T("ui.online") + "[-]"
	} else {
		out += "[" + config.Config.Colors.Negative + "]" + config.T("ui.offline") + "[-]"
	}
	out += " "
	out += "[::d] ("
	out += fmt.Sprint(statusInfo.BatteryCharge)
	out += "%"
	if statusInfo.BatteryLoading {
		out += " [" + config.Config.Colors.Positive + "]L[-]"
	} else {
		out += " [" + config.Config.Colors.Negative + "]l[-]"
	}
	if statusInfo.BatteryPowersave {
		out += " [" + config.Config.Colors.Negative + "]S[-]"
	} else {
		out += " [" + config.Config.Colors.Positive + "]s[-]"
	}
	out += ")[::-] "
	out += statusInfo.LastSeen
	infoBar.SetText(out)
	//infoBar.SetText("🔋: ??%")
}

// sets the current chat, loads text from storage to TextView
func SetDisplayedChat(wid messages.Chat) {
	//TODO: how to get chat to set
	currentReceiver = wid
	textView.Clear()
	linkLogReset()
	textView.SetTitle(wid.Name)
	sessionManager.CommandChannel <- messages.Command{"select", []string{currentReceiver.Id}}
}

// get a string representation of all messages for chat
func getMessagesString(msgs []messages.Message) string {
	out := ""
	for i := range msgs {
		out += getTextMessageString(&msgs[i], mediaTagFor(&msgs[i]))
		out += "\n"
	}
	return out
}

// mediaTagFor returns the per-chat attachment number marker ("[#N] ") for a
// media message, assigning it the next number in its chat. Text messages get
// no marker. Every attachment in the loaded chat gets a number; /show N opens
// the N-th one with the system viewer.
func mediaTagFor(msg *messages.Message) string {
	switch msg.Kind {
	case messages.MessageKindImage,
		messages.MessageKindVideo,
		messages.MessageKindAudio,
		messages.MessageKindDocument:
	default:
		return ""
	}
	n := assignMediaNumber(msg.ChatId, msg.Id)
	return "[::b][#" + strconv.Itoa(n) + "][::-] "
}

// assignMediaNumber records msgID as the next attachment of the chat and
// returns its 1-based number. The numbering restarts per chat and per screen
// build.
func assignMediaNumber(chatID, msgID string) int {
	mediaIndexByChat[chatID] = append(mediaIndexByChat[chatID], msgID)
	return len(mediaIndexByChat[chatID])
}

// resetChatMediaIndex starts the attachment numbering of a chat again at #1,
// which happens whenever its screen is rebuilt.
func resetChatMediaIndex(chatID string) {
	mediaIndexByChat[chatID] = nil
}

// resolveMediaNumber returns the message id of the N-th attachment shown in a
// chat.
func resolveMediaNumber(chatID string, n int) (string, bool) {
	idx := mediaIndexByChat[chatID]
	if n < 1 || n > len(idx) {
		return "", false
	}
	return idx[n-1], true
}

// create a formatted string with regions based on message ID from a text message
// TODO: optimize, use Sprintf etc
func getTextMessageString(msg *messages.Message, tag string) string {
	colorMe := config.Config.Colors.ChatMe
	colorContact := config.Config.Colors.ChatContact
	out := ""
	text := colorizeLinks(tview.Escape(msg.Text))
	if msg.Forwarded {
		text = "[" + config.Config.Colors.ForwardedText + "]" + text + "[-]"
	}
	tim := time.Unix(int64(msg.Timestamp), 0)
	time := tim.Format("02-01-06 15:04:05")
	out += "[\""
	out += msg.Id
	out += "\"]"
	if msg.FromMe { //msg from me
		out += "[-::d](" + time + ") [" + colorMe + "::b]" + config.T("ui.me") + " [-::-]" + tag + text
	} else { // message from others
		out += "[-::d](" + time + ") [" + colorContact + "::b]" + msg.ContactShort + ": [-::-]" + tag + text
	}
	out += "[\"\"]"
	return out
}

type UiHandler struct{}

func (u UiHandler) NewMessage(msg messages.Message) {
	//TODO: its stupid to "go" this as its supposed to run
	//on the ui thread anyway. But QueueUpdate blocks...?
	go app.QueueUpdateDraw(func() {
		curRegions = append(curRegions, msg)
		PrintText(getTextMessageString(&msg, mediaTagFor(&msg)))
	})
}

func (u UiHandler) NewScreen(msgs []messages.Message) {
	go app.QueueUpdateDraw(func() {
		textView.Clear()
		linkLogReset()
		resetChatMediaIndex(currentReceiver.Id)
		screen := getMessagesString(msgs)
		textView.SetText(screen)
		linkLogAppendText(screen)
		curRegions = msgs
		if screen == "" {
			if currentReceiver.Id == "" {
				renderHelpOverlay(PrintHelp)
			} else {
				PrintText("[::d] " + fmt.Sprintf(config.T("ui.no_messages"), config.Config.Keymap.CommandBacklog) + " [::-]")
			}
		}
	})
}

// loads the chat data from storage to the TreeView.
// Private chats (DMs) go under "Chats", groups (@g.us) under "Grupos".
func (u UiHandler) SetChats(ids []messages.Chat) {
	go app.QueueUpdateDraw(func() {
		chatRoot.ClearChildren()
		groupRoot.ClearChildren()
		oldId := currentReceiver.Id
		for _, element := range ids {
			name := element.Name
			if name == "" {
				name = strings.TrimSuffix(strings.TrimSuffix(element.Id, messages.GROUPSUFFIX), messages.CONTACTSUFFIX)
			}
			if element.Unread > 0 {
				name += " ([" + config.Config.Colors.UnreadCount + "]" + fmt.Sprint(element.Unread) + "[-])"
				//tim := time.Unix(element.LastMessage, 0)
				//sin := time.Since(tim)
				//since := fmt.Sprintf("%s", sin)
				//time := tim.Format("02-01-06 15:04:05")
				//name += since
			}
			node := tview.NewTreeNode(name).
				SetReference(element).
				SetSelectable(true)
			if element.IsGroup {
				node.SetColor(tcell.ColorNames[config.Config.Colors.ListGroup])
				// store new currentReceiver, else the selection on the left goes off
				if element.Id == oldId {
					currentReceiver = element
				}
				groupRoot.AddChild(node)
				if element.Id == currentReceiver.Id {
					treeView.SetCurrentNode(node)
				}
			} else {
				node.SetColor(tcell.ColorNames[config.Config.Colors.ListContact])
				// store new currentReceiver, else the selection on the left goes off
				if element.Id == oldId {
					currentReceiver = element
				}
				chatRoot.AddChild(node)
				if element.Id == currentReceiver.Id {
					treeView.SetCurrentNode(node)
				}
			}
		}
	})
}

func (u UiHandler) SetContacts(contacts []messages.Contact) {
	go app.QueueUpdateDraw(func() {
		contactRoot.ClearChildren()
		contactList = contacts
		for _, contact := range contacts {
			name := contact.Name
			if name == "" {
				name = contact.Short
			}
			if name == "" {
				name = contact.Id
			}
			node := tview.NewTreeNode(name).
				SetReference(contact).
				SetSelectable(true)
			node.SetColor(tcell.ColorNames[config.Config.Colors.ListContact])
			contactRoot.AddChild(node)
		}
	})
}

func (u UiHandler) SetStatuses(statuses []messages.Message) {
	go app.QueueUpdateDraw(func() {
		statusRoot.ClearChildren()
		for _, status := range statuses {
			title := status.ContactName
			if title == "" {
				title = status.SenderId
			}
			if title == "" {
				title = config.T("ui.estado")
			}
			if status.Text != "" {
				title += ": " + status.Text
			} else {
				title += " " + config.T("ui.multimedia")
			}
			node := tview.NewTreeNode(title).
				SetReference(status).
				SetSelectable(true)
			node.SetColor(tcell.ColorNames[config.Config.Colors.ListHeader])
			statusRoot.AddChild(node)
		}
	})
}

// RefreshLanguage re-renders the translated tree headers and re-prints the
// help/commands screens after a /lang change, so the UI adjusts immediately.
func (u UiHandler) RefreshLanguage() {
	go app.QueueUpdateDraw(func() {
		chatRoot.SetText(config.T("ui.chats"))
		groupRoot.SetText(config.T("ui.groups"))
		statusRoot.SetText(config.T("ui.statuses"))
		contactRoot.SetText(config.T("ui.contacts"))
		renderHelpOverlay(PrintHelp, PrintCommands)
	})
}

func (u UiHandler) PrintError(err error) {
	PrintError(err)
}

func (u UiHandler) PrintText(msg string) {
	PrintText(msg)
}

func (u UiHandler) OpenFile(path string) {
	open.Run(path)
}

func (u UiHandler) SetStatus(status messages.SessionStatus) {
	go app.QueueUpdateDraw(func() {
		UpdateStatusBar(status)
	})
}

// SetQRCode shows the pairing QR code as a message at the end of the chat
// history, so it never covers the other text. Every refresh replaces the
// previous QR block in place (WhatsApp rotates the code roughly every 20s);
// an empty string just removes it once pairing finishes.
func (u UiHandler) SetQRCode(ansi string) {
	go app.QueueUpdateDraw(func() {
		if ansi != "" {
			var buf strings.Builder
			if _, err := tview.ANSIWriter(&buf).Write([]byte(ansi)); err != nil {
				PrintError(err)
				return
			}
			ansi = buf.String()
		}
		newText, newLast := applyQRText(textView.GetText(false), lastQRText, ansi)
		lastQRText = newLast
		textView.SetText(newText)
		textView.ScrollToEnd()
	})
}

// applyQRText removes the previous QR block from the chat body (when there is
// one) and appends the new one below it. With an empty newQR it only removes
// the block. It returns the new body and the text to remember as the current
// QR block.
func applyQRText(body, lastQR, newQR string) (newBody, newLast string) {
	if lastQR != "" {
		if idx := strings.LastIndex(body, lastQR); idx >= 0 {
			body = body[:idx] + body[idx+len(lastQR):]
			// drop the separator newline we added before the block
			if idx > 0 && body[idx-1] == '\n' {
				body = body[:idx-1] + body[idx:]
			}
		}
	}
	if newQR == "" {
		return body, ""
	}
	if body == "" {
		return newQR, newQR
	}
	return body + "\n" + newQR, newQR
}
