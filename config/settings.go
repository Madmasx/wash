package config

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/adrg/xdg"
	"gopkg.in/ini.v1"
)

// AppFolder is the subdirectory created inside DownloadPath/PreviewPath where
// attachments land, so downloads do not get scattered loose in the base dir.
const AppFolder = "wash"

var configFilePath string
var cfg *ini.File

type IniFile struct {
	*General
	*Keymap
	*Ui
	*Colors
}

type General struct {
	DownloadPath        string
	PreviewPath         string
	CmdPrefix           string
	EnableNotifications bool
	UseTerminalBell     bool
	NotificationTimeout int64
	BacklogMsgQuantity  int
	Language            string
}

type Keymap struct {
	SwitchPanels    string
	FocusMessages   string
	FocusInput      string
	FocusChats      string
	Copyuser        string
	Pasteuser       string
	CommandBacklog  string
	CommandRead     string
	CommandConnect  string
	CommandQuit     string
	CommandHelp     string
	MessageDownload string
	MessageOpen     string
	MessageShow     string
	MessageUrl      string
	MessageInfo     string
	MessageRevoke   string
}

type Ui struct {
	ChatSidebarWidth int
}

type Colors struct {
	Background      string
	Text            string
	ForwardedText   string
	ListHeader      string
	ListContact     string
	ListGroup       string
	ChatContact     string
	ChatMe          string
	LinkColor       string
	Borders         string
	InputBackground string
	InputText       string
	UnreadCount     string
	Positive        string
	Negative        string
}

var Config = IniFile{
	&General{
		DownloadPath:        GetHomeDir() + "Downloads",
		PreviewPath:         GetHomeDir() + "Downloads",
		CmdPrefix:           "/",
		EnableNotifications: false,
		UseTerminalBell:     false,
		NotificationTimeout: 60,
		BacklogMsgQuantity:  10,
		Language:            "es",
	},
	&Keymap{
		SwitchPanels:    "Tab",
		FocusMessages:   "Ctrl+w",
		FocusInput:      "Ctrl+Space",
		FocusChats:      "Ctrl+e",
		CommandBacklog:  "Ctrl+b",
		CommandRead:     "Ctrl+n",
		Copyuser:        "Ctrl+c",
		Pasteuser:       "Ctrl+v",
		CommandConnect:  "Ctrl+r",
		CommandQuit:     "Ctrl+q",
		CommandHelp:     "Ctrl+p",
		MessageDownload: "d",
		MessageInfo:     "i",
		MessageOpen:     "o",
		MessageUrl:      "u",
		MessageRevoke:   "r",
		MessageShow:     "s",
	},
	&Ui{
		ChatSidebarWidth: 30,
	},
	&Colors{
		Background:      "default",
		Text:            "white",
		ForwardedText:   "purple",
		ListHeader:      "yellow",
		ListContact:     "green",
		ListGroup:       "cyan",
		ChatContact:     "green",
		ChatMe:          "cyan",
		LinkColor:       "lightblue",
		Borders:         "purple",
		InputBackground: "default",
		InputText:       "white",
		UnreadCount:     "yellow",
		Positive:        "green",
		Negative:        "red",
	},
}

func InitConfig() {
	var err error
	if configFilePath, err = xdg.ConfigFile("wash/wash.config"); err == nil {
		// add any new values
		var cfg *ini.File
		if cfg, err = ini.Load(configFilePath); err == nil {
			cfg.NameMapper = ini.TitleUnderscore
			cfg.ValueMapper = os.ExpandEnv
			if section, err := cfg.GetSection("general"); err == nil {
				section.MapTo(&Config.General)
			}
			if section, err := cfg.GetSection("keymap"); err == nil {
				section.MapTo(&Config.Keymap)
			}
			if section, err := cfg.GetSection("ui"); err == nil {
				section.MapTo(&Config.Ui)
			}
			if section, err := cfg.GetSection("colors"); err == nil {
				section.MapTo(&Config.Colors)
			}
			//TODO: only save if changes
			//newCfg := ini.Empty()
			//if err = ini.ReflectFromWithMapper(newCfg, &Config, ini.TitleUnderscore); err == nil {
			//err = newCfg.SaveTo(configFilePath)
			//}
		} else {
			cfg = ini.Empty()
			cfg.NameMapper = ini.TitleUnderscore
			cfg.ValueMapper = os.ExpandEnv
			if err = ini.ReflectFromWithMapper(cfg, &Config, ini.TitleUnderscore); err == nil {
				err = cfg.SaveTo(configFilePath)
			}
		}
	}
	if err != nil {
		fmt.Print(err.Error())
	}
}

func GetConfigFilePath() string {
	return configFilePath
}

// AbbreviateHome rewrites a path under the user's home directory to use ~.
// It is only for display: the UI prints these paths, and the splash screen is
// the first thing a screenshot of WaSh shows, so spelling out the account name
// leaks it into anything the user shares. Paths outside the home directory
// (a custom XDG_CONFIG_HOME, for instance) are returned untouched.
func AbbreviateHome(path string) string {
	// GetHomeDir carries a trailing separator, but do not rely on it.
	home := strings.TrimSuffix(GetHomeDir(), string(os.PathSeparator))
	if home == "" || home == path || !strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return path
	}
	return "~" + path[len(home):]
}

// Save persists the current config to the ini file (e.g. after changing
// settings such as the language at runtime).
func Save() error {
	if configFilePath == "" {
		return fmt.Errorf("config file path not initialized")
	}
	cfg := ini.Empty()
	cfg.NameMapper = ini.TitleUnderscore
	cfg.ValueMapper = os.ExpandEnv
	if err := ini.ReflectFromWithMapper(cfg, &Config, ini.TitleUnderscore); err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}
	return cfg.SaveTo(configFilePath)
}

func GetSessionFilePath() string {
	if sessionFilePath, err := xdg.ConfigFile("wash/session"); err == nil {
		return sessionFilePath
	}
	return GetHomeDir() + ".wash.session"
}

// gets the OS home dir with a path separator at the end
func GetHomeDir() string {
	usr, err := user.Current()
	if err != nil {
	}
	return usr.HomeDir + string(os.PathSeparator)
}
