# Project: "Booper" - Implementation Guide

Welcome to the team! "Booper" is an ambient, multiplayer "poke" game. This guide will walk you through the full implementation, from the Go backend to the JavaScript frontend.

Your goal is to create a web app where players' avatars (from Dicebear) bounce around the screen like an old school DVD screensaver. Players can click (or "boop") each other. The core mechanic is a "poke-back" system: if someone boops you, their avatar glows until you boop them back.

We're not writing any code for you, but this guide provides the complete architecture, data structures, and function-level logic you'll need to build from scratch.

## 1. Project Structure

First, set up this file structure. The Go server will live in the root, and all frontend assets will be in a `frontend` directory.

```
/booper/
|
|-- go.mod
|-- go.sum
|
|-- main.go           # Entry point, starts HTTP server
|-- handlers.go       # HTTP request handlers (serving files, WebSocket upgrade)
|-- hub.go            # WebSocket hub (manages all active connections)
|-- client.go         # Represents a single WebSocket client
|-- game.go           # Core game state (players, boop log)
|-- player.go         # `Player` struct and name generation
|
|-- /frontend/
|   |
|   |-- /templates/
|   |   |-- index.gohtml    # The main Go HTML template for our app
|   |
|   |-- /static/
|       |-- /css/
|       |   |-- style.css   # All frontend styles
|       |
|       |-- /js/
|       |   |-- main.js     # Core frontend logic (game loop, DOM manipulation)
|       |   |-- network.js  # WebSocket connection and message handling
|       |
|       |-- /sounds/
|           |-- booped.mp3    # Sound for when you get booped
|           |-- boop.mp3      # Sound for when you boop someone

```

## 2. Backend (Go)

The backend's job is to:

1. Serve the `index.gohtml` file and static assets.

2. Manage WebSocket connections.

3. Act as the "single source of truth" for player identity (names) and the "boop log."

4. Broadcast changes (like new players or new boops) to all connected clients.

### Step 2.1: Data Structures

You'll need these Go structs.

**`player.go`**

* **`Player` struct:**

  * `ID string`: A unique ID (e.g., `uuid.New().String()`).

  * `Name string`: A randomly generated "Adjective Animal" name (e.g., "Silly Wombat").

* **`NewPlayer()` function:**

  * Generates the random name (e.g., from two `[]string` slices).

  * Creates and returns a new `Player` with a new `ID` and the generated `Name`.

**`game.go`**

* **`GameState` struct:** This is the heart of the server.

  * `Players map[string]*Player`: A map of `player.ID` to the `Player` object.

  * `BoopLog map[string]map[string]bool`: A "who-booped-who" log.

    * The outer map key is the `booperID`.

    * The inner map is a "set" of `boopedID`s.

    * Example: `BoopLog["id-A"]["id-B"] = true` means "Player A booped Player B."

* **`Game` struct:** A singleton to manage the `GameState` and `Hub`.

  * `State *GameState`

  * `Hub *Hub`

* You will need to use `sync.Mutex` inside `GameState` to protect `Players` and `BoopLog` from concurrent read/writes.

**`hub.go` & `client.go`**

* This is a standard Go WebSocket hub implementation.

* **`Hub` struct:**

  * `clients map[*Client]bool`: All active clients.

  * `broadcast chan []byte`: Channel for messages to all clients.

  * `register chan *Client`: Channel to add a new client.

  * `unregister chan *Client`: Channel to remove a client.

  * `game *Game`: A reference back to the main game.

* **`Client` struct:**

  * `hub *Hub`

  * `conn *websocket.Conn`: The connection itself.

  * `send chan []byte`: A message buffer for this client.

  * `playerID string`: The ID of the player this client represents.

* The `Hub` needs a `Run()` method (in a goroutine) to `select` on its channels.

* The `Client` needs `readPump()` and `writePump()` methods (in goroutines) to handle messages.

### Step 2.2: WebSocket "API" (Message Protocol)

This is the most critical part. Define a rigid JSON structure for all server-client communication.

**Base Message Wrapper:**

```
{ "type": "message_type", "payload": { ... } }

```

#### Server-to-Client (S2C) Messages

1. **`welcome`** (Sent *only* to a new client on connection)

   * Tells the new client their own identity and the *entire* current game state.

   * `type`: "welcome"

   * `payload`: `{ "self": Player, "currentState": GameState }`

2. **`player_joined`** (Sent to *all other* clients)

   * `type`: "player_joined"

   * `payload`: `Player` (the new player who just connected)

3. **`player_left`** (Sent to *all* remaining clients)

   * `type`: "player_left"

   * `payload`: `{ "id": "player-id-who-left" }`

4. **`boop_event`** (Sent to *all* clients when a boop happens)

   * `type`: "boop_event"

   * `payload`: `{ "booperID": "id-of-booper", "boopedID": "id-of-booped" }`

#### Client-to-Server (C2S) Messages

1. **`boop_request`** (Sent when a client clicks an avatar)

   * `type`: "boop_request"

   * `payload`: `{ "boopedID": "id-of-player-i-clicked" }`

### Step 2.3: Handler Logic

**`handlers.go`**

* **`HandleRoot` (`/`)**:

  * Parses the `frontend/templates/index.gohtml` template.

  * Executes and serves it. (No initial data needs to be injected; the WebSocket will provide it).

* **`HandleStatic` (`/static/`)**:

  * Uses `http.StripPrefix` and `http.FileServer` to serve the `frontend/static` directory.

* **`HandleWebSocket` (`/ws`)**:

  * Upgrades the HTTP connection to a WebSocket.

  * Creates a new `Player` object for this connection.

  * Adds the `Player` to the `Game.State.Players`.

  * Creates a new `Client`, assigns the `player.ID` to it, and registers it with the `Hub`.

  * Sends the `welcome` message to the new client.

  * Broadcasts a `player_joined` message to all *other* clients.

**`client.go` (in `readPump`)**

* This is where you handle C2S messages.

* Listen for `boop_request` messages.

* When one arrives:

  1. Get the `boopedID` from the payload.

  2. The `booperID` is the `client.playerID`.

  3. Lock the `Game.State`.

  4. Update the `Game.State.BoopLog`: `BoopLog[booperID][boopedID] = true`.

  5. Unlock.

  6. Broadcast the new `boop_event` to *all* clients (including the sender).

## 3. Frontend (HTML, CSS, JS)

### Step 3.1: HTML & CSS

**`frontend/templates/index.gohtml`**

* Basic HTML5 boilerplate.

* The `<body>` should contain:

  * `<div id="game-world"></div>` (This is the "screen" where avatars bounce).

  * `<audio id="sound-boop" src="/static/sounds/boop.mp3"></audio>`

  * `<audio id="sound-booped" src="/static/sounds/booped.mp3"></audio>`

* Include your scripts at the end of the `<body>`:

  * `<script src="/static/js/network.js"></script>`

  * `<script src="/static/js/main.js"></script>`

**`frontend/static/css/style.css`**

* `body`: `margin: 0; overflow: hidden;`

* `#game-world`:

  * `position: relative;`

  * `width: 100vw; height: 100vh;`

  * `background-color: #222;`

* `.avatar`: (This class will be for the avatar `<img>` elements)

  * `position: absolute;`

  * `width: 80px; height: 80px;` (or whatever size)

  * `border-radius: 50%;`

  * `cursor: pointer;`

  * `transition: box-shadow 0.3s ease;`

* `.avatar.glowing`:

  * `box-shadow: 0 0 15px 5px #00ffff;` (This is the aura!)

### Step 3.2: JavaScript - `network.js`

This file's only job is to talk to the server.

* Create a `WebSocket` connection: `const socket = new WebSocket("ws://" + window.location.host + "/ws");`

* **`socket.onmessage`:**

  * Parse the incoming event `data` (it's a JSON string).

  * Use a `switch (message.type)` statement.

  * Call functions in `main.js` based on the type.

    * `case "welcome": handleWelcome(message.payload); break;`

    * `case "player_joined": handlePlayerJoined(message.payload); break;`

    * `case "player_left": handlePlayerLeft(message.payload); break;`

    * `case "boop_event": handleBoopEvent(message.payload); break;`

* **`sendBoop(boopedID)` function:**

  * This is the *only* C2S function.

  * It constructs the `boop_request` JSON object and sends it:

  * `socket.send(JSON.stringify({ type: "boop_request", payload: { boopedID: boopedID } }));`

### Step 3.3: JavaScript - `main.js`

This is the client-side game!

* **Global State Variables:**

  * `let myID = "";`

  * `let players = {};` (This is the *client-side* player store)

  * `let boopLog = {};`

  * Get DOM elements: `const gameWorld = document.getElementById("game-world");`

* **The `players` Object:**

  * This is the key to the simulation. It will store player data *and* simulation data.

  * `players["id-A"] = {`

  * `  id: "id-A",`

  * `  name: "Silly Wombat",`

  * `  domElement: <HTMLImageElement>,`

  * `  x: 150, y: 200,` (current position)

  * `  vx: 2, vy: 1` (current velocity)

  * `}`

* **Handler Functions (called by `network.js`):**

  * **`handleWelcome(payload)`:**

    1. `myID = payload.self.id;`

    2. `boopLog = payload.currentState.boopLog;`

    3. Loop through `payload.currentState.players`: `for (const id in payload.currentState.players) { ... }`

    4. For each, call `createPlayerAvatar(payload.currentState.players[id]);`

    5. `updateAllAuras();` (See function below)

  * **`handlePlayerJoined(playerData)`:**

    1. Call `createPlayerAvatar(playerData);`

  * **`handlePlayerLeft(payload)`:**

    1. Find the player in your `players` map: `const player = players[payload.id];`

    2. Remove their `player.domElement` from the `gameWorld`.

    3. `delete players[payload.id];`

  * **`handleBoopEvent(payload)`:**

    1. Update your local `boopLog`.

    2. `if (!boopLog[payload.booperID]) boopLog[payload.booperID] = {};`

    3. `boopLog[payload.booperID][payload.boopedID] = true;`

    4. Play sounds:

       * `if (payload.boopedID === myID) playSound("sound-booped");`

       * `else if (payload.booperID === myID) playSound("sound-boop");`

    5. `updateAllAuras();` (This re-checks who needs to glow)

* **Core Game Functions:**

  * **`createPlayerAvatar(playerData)`:**

    1. `const img = document.createElement("img");`

    2. `img.className = "avatar";`

    3. Set the Dicebear URL: `img.src = \`https://www.google.com/search?q=https://api.dicebear.com/8.x/adventurer/svg%3Fseed%3D%24{playerData.name}\`;\`

    4. `img.onclick = () => { network.sendBoop(playerData.id); };`

    5. `gameWorld.appendChild(img);`

    6. **Initialize DVD screensaver simulation:**

       * `const x = Math.random() * (gameWorld.clientWidth - 80);`

       * `const y = Math.random() * (gameWorld.clientHeight - 80);`

       * `const vx = (Math.random() - 0.5) * 4;` (random speed/direction)

       * `const vy = (Math.random() - 0.5) * 4;`

    7. Store everything in the `players` map:

       * `players[playerData.id] = { id: playerData.id, name: playerData.name, domElement: img, x, y, vx, vy };`

  * **`updateAllAuras()`:**

    1. Loop through all players: `for (const id in players) { ... }`

    2. `const player = players[id];`

    3. **This is the core "poke-back" logic:**

    4. Check: "Has this player booped me?"

       * `const theyBoopedMe = boopLog[player.id] && boopLog[player.id][myID];`

    5. Check: "Have I booped them back?"

       * `const iBoopedThem = boopLog[myID] && boopLog[myID][player.id];`

    6. **Set the glow:**

       * `if (theyBoopedMe && !iBoopedThem) { player.domElement.classList.add("glowing"); }`

       * `else { player.domElement.classList.remove("glowing"); }`

  * **`gameLoop()` (The simulation):**

    1. Loop through all players: `for (const id in players) { ... }`

    2. `const p = players[id];`

    3. `p.x += p.vx; p.y += p.vy;`

    4. **Collision detection:**

       * `if (p.x <= 0 || p.x + 80 >= gameWorld.clientWidth) p.vx *= -1;`

       * `if (p.y <= 0 || p.y + 80 >= gameWorld.clientHeight) p.vy *= -1;`

    5. Move the element on screen: `p.domElement.style.transform = \`translate(\${p.x}px, ${p.y}px)\`;\`

  * **Start the loop:**

    * `function runGame() { gameLoop(); requestAnimationFrame(runGame); }`

    * `runGame();`