package config

// translations holds the UI strings per language. The language is selected
// with the `language` key under [general] in wash.config (or `/lang` at
// runtime). If a key is missing for the active language, it falls back to
// English, then to the raw key.
var translations = map[string]map[string]string{
	"es": {
		// Tree
		"ui.chats":      "Chats",
		"ui.groups":     "Grupos",
		"ui.statuses":   "Estados",
		"ui.contacts":   "Contactos",
		"ui.estado":     "Estado",
		"ui.multimedia": "[Multimedia]",

		// Help screen
		"help.keys":          "Teclas:",
		"help.global":        "Global",
		"help.scroll":        "Arriba/Abajo = Desplazar historial/chats",
		"help.switch_input":  "Alternar entrada/chats",
		"help.focus_msg":     "Enfocar panel de mensajes",
		"help.exit":          "Salir de la app",
		"help.lang":          "Cambiar el idioma de la interfaz (es | en)",
		"help.dragdrop":      "Suelta un archivo en la ventana y presiona Enter para enviarlo como adjunto",
		"help.mention":       "Escribe @Nombre para abrir conversación directa (con sugerencias)",
		"help.msg_panel":     "Panel de mensajes",
		"help.select_msg":    "seleccionar mensaje",
		"help.download":      "Descargar adjunto",
		"help.open":          "Descargar y abrir adjunto",
		"help.show":          "Descargar y abrir adjunto con el visor del sistema",
		"help.show_index":    "los adjuntos se numeran [#N] por chat; /show N abre el #N",
		"help.url":           "Buscar URL en el mensaje y abrirla",
		"help.click_link":    "Haz clic en un enlace para abrirlo en tu navegador",
		"help.revoke":        "Revocar mensaje",
		"help.info":          "Información del mensaje",
		"help.config_file":   "Archivo de config en ->",
		"help.type_commands": "Escribe [::b]%s[::-] para ver todos los comandos",

		// Commands screen
		"cmds.commands":    "Comandos:",
		"cmds.global":      "Global",
		"cmds.connect":     "(Re)conectar al servidor",
		"cmds.disconnect":  "Cerrar la conexión",
		"cmds.logout":      "Eliminar datos de inicio de sesión del equipo",
		"cmds.reset":       "Borrar sesión guardada y reconectar limpiamente",
		"cmds.quit":        "Salir de la app",
		"cmds.lang":        "Cambiar el idioma de la interfaz (es | en)",
		"cmds.chat":        "Chat",
		"cmds.backlog":     "cargar los siguientes %d mensajes anteriores",
		"cmds.read":        "marcar como leídos los mensajes nuevos del chat",
		"cmds.upload":      "Subir cualquier archivo como documento",
		"cmds.sendimage":   "Enviar mensaje de imagen",
		"cmds.sendvideo":   "Enviar mensaje de video",
		"cmds.sendaudio":   "Enviar mensaje de audio",
		"cmds.show":        "abrir el adjunto #N (numerado [#N] en el chat, N o message-id) con tu visor del sistema",
		"cmds.groups":      "Grupos",
		"cmds.leave":       "Salir del grupo",
		"cmds.create":      "Crear grupo con usuarios",
		"cmds.subject":     "Cambiar asunto del grupo",
		"cmds.add":         "Agregar usuario al grupo",
		"cmds.remove":      "Quitar usuario del grupo",
		"cmds.admin":       "Asignar rol de admin a usuario del grupo",
		"cmds.removeadmin": "Quitar rol de admin a usuario del grupo",
		"cmds.copyid":      "para copiar el id de usuario seleccionado al portapapeles",
		"cmds.paste":       "para pegar el portapapeles en la entrada de texto (o enviar una captura/imagen del portapapeles)",

		// General UI
		"ui.no_receiver":       "sin receptor seleccionado",
		"ui.contact_not_found": "Contacto no encontrado:",
		"ui.paste_image_sent":  "📷 Imagen del portapapeles enviada a",
		"ui.link_open":         "Abriendo enlace:",
		"ui.online":            "en línea",
		"ui.offline":           "sin conexión",
		"ui.me":                "Yo:",
		"ui.no_messages":       "~~~ sin mensajes, presiona %s para cargar historial si está disponible ~~~",

		// Session / connection feedback
		"session.connecting":    "conectando..",
		"session.connected":     "conectado",
		"session.disconnected":  "desconectado",
		"session.closing":       "cerrando el receptor",
		"session.loggedout":     "Sesión cerrada: ",
		"session.loggedout_ok":  "Sesión cerrada correctamente",
		"session.unknown_cmd":   "Comando desconocido: ",
		"session.conn_failed":   "Falló la conexión a WhatsApp: ",
		"session.try_reset":     "Prueba con /reset para restablecer la conexión por completo",
		"session.connected_ok":  "Conectado a WhatsApp correctamente",
		"session.not_connected": "no conectado a WhatsApp",
		"session.retrieving":    "Recuperando historial de mensajes...",
		"session.no_anchor":     "Aún no hay ancla local de mensajes. Abre el chat después de que la sincronización de WhatsApp entregue historial y vuelve a intentar con /backlog.",
		"session.logout_warn":   "Advertencia: no se pudo cerrar sesión por completo: ",
		"session.invalid_jid":   "JID inválido: ",

		// /show N (attachment index)
		"show.no_index": "no hay adjunto #%d en este chat",

		// /lang command
		"lang.usage":       "Uso: /lang es | /lang en",
		"lang.unsupported": "Idioma no soportado. Usa: /lang es | /lang en",
		"lang.set":         "Idioma configurado a: %s",
	},
	"en": {
		// Tree
		"ui.chats":      "Chats",
		"ui.groups":     "Groups",
		"ui.statuses":   "Statuses",
		"ui.contacts":   "Contacts",
		"ui.estado":     "Estado",
		"ui.multimedia": "[Multimedia]",

		// Help screen
		"help.keys":          "Keys:",
		"help.global":        "Global",
		"help.scroll":        "Up/Down = Scroll history/chats",
		"help.switch_input":  "Switch input/chats",
		"help.focus_msg":     "Focus message panel",
		"help.exit":          "Exit app",
		"help.lang":          "Change the UI language (es | en)",
		"help.dragdrop":      "Drop a file onto the window and press Enter to send it as an attachment",
		"help.mention":       "Type @Name to open a direct conversation (with suggestions)",
		"help.msg_panel":     "Message panel",
		"help.select_msg":    "select message",
		"help.download":      "Download attachment",
		"help.open":          "Download & open attachment",
		"help.show":          "Download & open attachment with your system viewer",
		"help.show_index":    "attachments are numbered [#N] per chat; /show N opens #N",
		"help.url":           "Find URL in message and open it",
		"help.click_link":    "Click a link to open it in your browser",
		"help.revoke":        "Revoke message",
		"help.info":          "Info about message",
		"help.config_file":   "Config file in ->",
		"help.type_commands": "Type [::b]%s[::-] to see all commands",

		// Commands screen
		"cmds.commands":    "Commands:",
		"cmds.global":      "Global",
		"cmds.connect":     "(Re)Connect to server",
		"cmds.disconnect":  "Close the connection",
		"cmds.logout":      "Remove login data from computer",
		"cmds.reset":       "Remove stored session and reconnect cleanly",
		"cmds.quit":        "Exit app",
		"cmds.lang":        "Change the UI language (es | en)",
		"cmds.chat":        "Chat",
		"cmds.backlog":     "load next %d previous messages",
		"cmds.read":        "mark new messages in chat as read",
		"cmds.upload":      "Upload any file as document",
		"cmds.sendimage":   "Send image message",
		"cmds.sendvideo":   "Send video message",
		"cmds.sendaudio":   "Send audio message",
		"cmds.show":        "open attachment #N (numbered [#N] in chat, N or message-id) with your system viewer",
		"cmds.groups":      "Groups",
		"cmds.leave":       "Leave group",
		"cmds.create":      "Create group with users",
		"cmds.subject":     "Change subject of group",
		"cmds.add":         "Add user to group",
		"cmds.remove":      "Remove user from group",
		"cmds.admin":       "Set admin role for user in group",
		"cmds.removeadmin": "Remove admin role for user in group",
		"cmds.copyid":      "to copy a selected user id to clipboard",
		"cmds.paste":       "to paste clipboard to text input (or send a clipboard screenshot/image)",

		// General UI
		"ui.no_receiver":       "no receiver",
		"ui.contact_not_found": "Contact not found:",
		"ui.paste_image_sent":  "📷 Clipboard image sent to",
		"ui.link_open":         "Opening link:",
		"ui.online":            "online",
		"ui.offline":           "offline",
		"ui.me":                "Me:",
		"ui.no_messages":       "~~~ no messages, press %s to load backlog if available ~~~",

		// Session / connection feedback
		"session.connecting":    "connecting..",
		"session.connected":     "connected",
		"session.disconnected":  "disconnected",
		"session.closing":       "closing the receiver",
		"session.loggedout":     "Logged out: ",
		"session.loggedout_ok":  "Successfully logged out",
		"session.unknown_cmd":   "Unknown command: ",
		"session.conn_failed":   "WhatsApp connection failed: ",
		"session.try_reset":     "Try using /reset to completely reset the connection",
		"session.connected_ok":  "Successfully connected to WhatsApp",
		"session.not_connected": "not connected to WhatsApp",
		"session.retrieving":    "Retrieving message history...",
		"session.no_anchor":     "No local message anchor found yet. Open the chat after WhatsApp sync delivers some history, then try /backlog again.",
		"session.logout_warn":   "Warning: Couldn't fully log out: ",
		"session.invalid_jid":   "invalid JID: ",

		// /show N (attachment index)
		"show.no_index": "no attachment #%d in this chat",

		// /lang command
		"lang.usage":       "Usage: /lang es | /lang en",
		"lang.unsupported": "Unsupported language. Use: /lang es | /lang en",
		"lang.set":         "Language set to: %s",
	},
}

// T returns the translated string for key in the configured language.
// Falls back to English and finally to the raw key.
func T(key string) string {
	lang := Config.General.Language
	if lang == "" {
		lang = "es"
	}
	if table, ok := translations[lang]; ok {
		if text, ok := table[key]; ok {
			return text
		}
	}
	if en, ok := translations["en"]; ok {
		if text, ok := en[key]; ok {
			return text
		}
	}
	return key
}
