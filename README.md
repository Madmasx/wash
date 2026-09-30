# 🐚 WaSh — WhatsApp desde tu terminal

> **v2.2.0** · Cliente de WhatsApp para terminal, construido sobre [go-whatsmeow](https://github.com/tulir/whatsmeow) y [tview](https://github.com/rivo/tview)

![WaSh screenshot](/doc/WaSh-v2.2.0.png?raw=true "WaSh v2.2.0")

WaSh (antes *whatscli*) es un cliente de WhatsApp que vive en tu terminal: se conecta a través de la API Web (sin navegador), se vincula escaneando un código QR y te deja chatear, mandar archivos y gestionar grupos con el teclado y una interfaz limpia en español.

---

## ✨ Características

### Chats y mensajes
- 💬 Envía y recibe mensajes de WhatsApp desde la terminal
- 🔗 Conexión por la API Web de WhatsApp, sin navegador
- 📱 Vinculación simple con código QR, mostrado como bloque al final del chat (nunca tapa el texto) — ¡y como caracteres half-block para que se vea completo!
- 🗂️ **Secciones separadas en el árbol**: `Chats` (conversaciones privadas), `Grupos` (@g.us), `Estados` y `Contactos`
- 📥 Descarga y abre adjuntos (imagen/video/audio/documento) con tu visor del sistema
- 🔢 **Adjuntos numerados `[#N]` por chat**: `/show N` abre el N-ésimo adjunto directo con tu visor, sin buscar su id
- 📤 Envía imágenes, videos, audios y documentos
- 🔔 Notificaciones de escritorio y campana de terminal

### 🚀 Trucos de productividad (los favoritos)
- **🖼️ Pegar capturas de pantalla**: toma una captura, presiona `Ctrl+V` en WaSh y se envía como imagen al chat actual (X11/Wayland, sin escribir nada)
- **📎 Arrastrar y soltar**: suelta un archivo sobre la ventana de la terminal (WezTerm, kitty…) y presiona `Enter` — WaSh detecta la ruta y lo envía como adjunto según su tipo
- **👤 Menciones `@contacto`**: escribe `@` y aparecen sugerencias de tu lista de contactos mientras escribes; navega con `↑/↓`, `Enter` para fijar y `Enter` para **abrir la conversación directa**. Dentro de un mensaje, `@Nombre` se convierte al formato real de WhatsApp (`@5215512345678`)
- **⌨️ Autocompletado de comandos**: escribe `/` y se listan todos los comandos; sigue escribiendo para filtrarlos (`/send` → `/sendimage`, `/sendvideo`…)
- **🖱️ Enlaces clicables**: las URLs de los mensajes se resaltan en color (configurable con `LinkColor` en `[colors]`) y un clic sobre ellas las abre directamente en tu navegador
- **🌐 Bilingüe**: interfaz en español (por defecto) o inglés, cambiable en caliente con `/lang es|en`
- 🎨 Colores totalmente personalizables (tema Dracula listo en la config)

### Gestión de grupos
- Crear grupos, cambiar asunto, agregar/remover participantes, asignar/quitar admin y salir del grupo

---

## 🖥️ Requisitos

- **Go ≥ 1.25** para compilar
- **xclip** (X11) o **wl-clipboard** (Wayland) para pegar capturas de pantalla
- Un terminal moderno con soporte de color (recomendado: [WezTerm](https://wezfurlong.org/wezterm/))

## ⚙️ Instalación

```bash
git clone https://github.com/madmasx/wash.git
cd wash
make build        # genera el binario ./wash
./wash            # escanea el QR con WhatsApp en tu teléfono
```

> 💡 Si no alcanzas a escanear el QR a tiempo, reinicia la app o agranda la ventana / reduce la fuente. Después del primer escaneo, WaSh inicia sesión solo.

## 🎮 Uso

Selecciona un chat en el árbol de la izquierda y escribe en el campo de abajo para enviar mensajes. Cambia entre la lista y el campo con `Tab`.

Escribe `/help` o `/commands` dentro de la app para ver toda la ayuda en pantalla.

### Comandos

| Comando | Descripción |
| --- | --- |
| `/send` | Enviar mensaje de texto |
| `/sendimage <ruta>` | Enviar imagen |
| `/sendvideo <ruta>` | Enviar video |
| `/sendaudio <ruta>` | Enviar audio |
| `/upload <ruta>` | Enviar cualquier archivo como documento |
| `/backlog` | Cargar mensajes anteriores del chat |
| `/read` | Marcar mensajes como leídos |
| `/download` · `/open` | Descargar, o descargar y abrir el mensaje seleccionado |
| `/show N` | Abrir el adjunto `#N` del chat (numerados `[#N]`) con tu visor del sistema |
| `/url` | Abrir la URL del mensaje seleccionado |
| `/revoke` | Revocar mensaje |
| `/create <ids> <asunto>` | Crear grupo |
| `/leave` | Salir del grupo actual |
| `/subject <nuevo>` | Cambiar asunto del grupo |
| `/add` · `/remove` · `/admin` · `/removeadmin` | Gestionar participantes |
| `/info` | Información del mensaje seleccionado |
| `/select <id>` | Abrir un chat por su id |
| `/colorlist` | Ver todos los colores disponibles |
| `/lang es\|en` | Cambiar idioma de la interfaz |
| `/connect` · `/disconnect` · `/reset` · `/logout` | Gestión de sesión |
| `/help` · `/commands` | Ayuda en pantalla |
| `/quit` | Salir |

> Las rutas no necesitan comillas, incluso con espacios: `/sendimage /home/user/Mis Fotos/vacaciones.png` funciona tal cual. También aceptan `~`.

### Envío de archivos sin escribir comandos

1. **Captura de pantalla** → `Ctrl+V` en WaSh → se envía como imagen. 📷
2. **Arrastra un archivo** a la ventana → `Enter` → se envía solo (imagen/video/audio/documento según extensión).
3. **`@Nombre`** → sugerencias → `Enter` → `Enter` → conversación directa con ese contacto.

### Menciones en grupos

Escribe `@` y verás sugerencias de tu lista de contactos. Al enviar un mensaje con `@Nombre`, la mención se convierte automáticamente al formato real de WhatsApp (`@<número>`), igual que en la app oficial.

### Notificaciones

Activa `enable_notifications = true` en `wash.config` para notificaciones de escritorio, o `use_terminal_bell = true` para la campana de la terminal.

## 🎨 Configuración

La configuración vive en `~/.config/wash/wash.config` (la ruta exacta la muestra `/help`). Puedes cambiar:

- **Idioma**: `language = es | en` bajo `[general]`
- **Colores**: sección `[colors]` — ejemplo de tema Dracula:
  ```ini
  [colors]
  background       = default
  text             = white
  forwarded_text   = purple
  list_header      = purple
  list_contact     = green
  list_group       = cyan
  chat_contact     = green
  chat_me          = magenta
  borders          = purple
  unread_count     = purple
  positive         = green
  negative         = red
  ```
- **Atajos de teclado**: sección `[keymap]` (foco, mensajes, copiar/pegar…)
- **Descargas**: `download_path` y `preview_path` — los adjuntos caen en `<ruta>/wash/`, no sueltos en la carpeta base.

## 🛠️ Desarrollo

```bash
make build          # compila ./wash
go test ./...       # tests unitarios (config, messages, main)
go run .            # ejecución en desarrollo
```

### Estructura

- **`main.go`** — toda la UI (tview): árbol de chats/grupos/estados/contactos, campo de entrada con autocompletado de comandos y menciones, drag & drop, pegado de capturas y gestión de mensajes.
- **`links.go`** — enlaces clicables: índice línea→URL de la vista de mensajes (replica el wrapping de tview para mapear el clic exacto) y apertura en el navegador.
- **`messages/`** — el núcleo: `session_manager.go` (rutina separada que drena los comandos de la UI y los eventos de whatsmeow vía canales), `storage.go` (base de datos SQLite), `messages.go` (estructuras e interfaces).
- **`config/`** — singleton de configuración (ini) y sistema de traducción es/en (`i18n.go`).
- **`qrcode/`** — renderizado del QR de inicio de sesión.

## 📜 Créditos

WaSh es un fork con mucho cariño de [whatscli](https://github.com/normen/whatscli) (MIT) — gracias a su autor original por la base. Las mejoras: interfaz en español, sección de grupos y estados, autocompletado de comandos y contactos, drag & drop, pegado de capturas, enlaces clicables y tema Dracula.

Hecho con ❤️, ☕ y mucha paciencia por **[@madmasx](https://github.com/madmasx)** 😎

## 📄 Licencia

MIT. Recuerda: la licencia te da libertad, pero no para borrar el crédito del proyecto original. 😉
