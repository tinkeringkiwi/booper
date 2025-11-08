// main.js - UI, game loop, and message handlers (classic script, uses global window.network)

(() => {
  const AVATAR_SIZE = 80;
  const SPEED = 2; // pixels per tick (frame). Tune as desired.

  let myID = '';
  let myName = '';
  const players = {}; // id -> { id, name, domElement, x, y, vx, vy }
  let boopLog = {}; // booperID -> { boopedID: true }
  let myBoopsMade = 0;
  let myBoopsReceived = 0;
  const boopLogEl = document.getElementById('boop-log');

  const gameWorld = document.getElementById('game-world');

  // Audio management: unlock on first interaction for autoplay policies
  const audio = {
    boop: document.getElementById('sound-boop'),
    booped: document.getElementById('sound-booped'),
    unlocked: false,
    unlockOnce() {
      if (this.unlocked) return;
      const unlock = () => {
        [this.boop, this.booped].forEach(a => {
          if (!a) return;
          try {
            a.volume = 0.7;
            // Prime the element so future programmatic plays succeed
            a.play().then(() => {
              a.pause();
              a.currentTime = 0;
            }).catch(() => {/* ignore */});
          } catch (_) {}
        });
        this.unlocked = true;
        window.removeEventListener('pointerdown', unlock);
        window.removeEventListener('keydown', unlock);
      };
      window.addEventListener('pointerdown', unlock, { once: true });
      window.addEventListener('keydown', unlock, { once: true });
    }
  };
  audio.unlockOnce();

  // Utilities
  function playSound(id){
    let el = null;
    if (id === 'sound-boop') el = audio.boop;
    else if (id === 'sound-booped') el = audio.booped;
    if (!el) return;
    try { el.currentTime = 0; el.play(); } catch(_){}
  }

  function updateStats(){
    const madeEl = document.getElementById('stat-made');
    const recEl = document.getElementById('stat-received');
    if (madeEl) madeEl.textContent = String(myBoopsMade);
    if (recEl) recEl.textContent = String(myBoopsReceived);
  }

  function dicebearUrl(name){
    const seed = encodeURIComponent(name || 'Player');
    return `https://api.dicebear.com/8.x/adventurer/svg?seed=${seed}`;
  }

  function addBoopLog(booperID, boopedID){
    if (!boopLogEl) return;
    // Resolve names and avatars; include self via myName
    const booper = (booperID === myID) ? { name: myName } : players[booperID];
    const booped = (boopedID === myID) ? { name: myName } : players[boopedID];
    const booperName = booper?.name || 'Player';
    const boopedName = booped?.name || 'Player';
    const booperAvatar = dicebearUrl(booperName);
    const boopedAvatar = dicebearUrl(boopedName);

    const entry = document.createElement('div');
    entry.className = 'boop-entry';
    entry.innerHTML = `
      <img src="${booperAvatar}" alt="${booperName}">
      <span class="boop-arrow">👉</span>
      <img src="${boopedAvatar}" alt="${boopedName}">
    `;
    boopLogEl.prepend(entry);
    // Auto fade and remove after a few seconds
    const lifetime = 3500;
    setTimeout(() => { entry.classList.add('fade-out'); }, lifetime - 400);
    setTimeout(() => { if (entry.parentNode) entry.parentNode.removeChild(entry); }, lifetime);
    // Keep at most N entries
    const maxEntries = 5;
    while (boopLogEl.children.length > maxEntries) boopLogEl.removeChild(boopLogEl.lastChild);
  }

  function createPlayerAvatar(playerData){
    if (playerData.id === myID) return; // don't render own avatar
    if (players[playerData.id]) return; // already exists
    const img = document.createElement('img');
    img.className = 'avatar';
    // Use direct Dicebear endpoint for avatars
    img.src = dicebearUrl(playerData.name);
    img.alt = playerData.name;
    img.title = playerData.name;
    img.draggable = false;
    img.onclick = () => {
      if (playerData.id === myID || !window.network) return;
      const iBoopedThem = !!(boopLog[myID] && boopLog[myID][playerData.id]);
      if (iBoopedThem) { window.handleBoopDenied && window.handleBoopDenied({ boopedID: playerData.id }); return; }
      window.network.sendBoop(playerData.id);
    };
    gameWorld.appendChild(img);

    const x = Math.random() * (gameWorld.clientWidth - AVATAR_SIZE);
    const y = Math.random() * (gameWorld.clientHeight - AVATAR_SIZE);
  // Initialize with a random direction but constant speed
  let angle = Math.random() * Math.PI * 2;
  let vx = Math.cos(angle) * SPEED;
  let vy = Math.sin(angle) * SPEED;

    players[playerData.id] = { id: playerData.id, name: playerData.name, domElement: img, x, y, vx, vy };
  }

  function updateAllAuras(){
    // Glow any avatar that has a current boop directed at me (latest direction retained server-side).
    for (const id in players){
      const player = players[id];
      const theyBoopedMe = !!(boopLog[player.id] && boopLog[player.id][myID]);
      // Mark avatars I have already booped and cannot re-boop until direction changes
      const iBoopedThem = !!(boopLog[myID] && boopLog[myID][id]);
      if (theyBoopedMe) player.domElement.classList.add('glowing');
      else player.domElement.classList.remove('glowing');
      if (iBoopedThem) player.domElement.classList.add('already-booped');
      else player.domElement.classList.remove('already-booped');
    }
  }

  function gameLoop(){
    for (const id in players){
      const p = players[id];
      p.x += p.vx; p.y += p.vy;
      // Wall collisions: invert direction and re-normalize to guarantee constant speed
      if (p.x <= 0 || p.x + AVATAR_SIZE >= gameWorld.clientWidth) {
        p.vx = -p.vx;
      }
      if (p.y <= 0 || p.y + AVATAR_SIZE >= gameWorld.clientHeight) {
        p.vy = -p.vy;
      }
      // Ensure exact speed (avoid drift from float ops)
      const mag = Math.hypot(p.vx, p.vy) || 1;
      p.vx = (p.vx / mag) * SPEED;
      p.vy = (p.vy / mag) * SPEED;
      p.domElement.style.transform = `translate(${p.x}px, ${p.y}px)`;
    }
  }
  function runGame(){
    gameLoop();
    requestAnimationFrame(runGame);
  }
  runGame();

  // Network handlers (exposed globally so network.js can call them)
  window.handleWelcome = (payload) => {
    // Version check: if server start changed, store and reload to get fresh assets
    const serverStart = payload.serverStart;
    try {
      const key = 'booper_server_start';
      const prev = localStorage.getItem(key);
      if (serverStart && prev && prev !== serverStart) {
        localStorage.setItem(key, serverStart); // set before reload to avoid loops
        location.reload();
        return;
      }
      if (serverStart && !prev) localStorage.setItem(key, serverStart);
    } catch (_) {}
    myID = payload.self.id;
    myName = payload.self.name || '';
    boopLog = payload.currentState.boopLog || {};
    // Initialize my stats if provided by server
    const madeMap = payload.currentState.boopsMade || {};
    const recMap = payload.currentState.boopsReceived || {};
    myBoopsMade = madeMap[myID] || 0;
    myBoopsReceived = recMap[myID] || 0;
    const statePlayers = payload.currentState.players || {};
    // Clear any existing avatars (for reconnections without reload)
    for (const id in players){
      const p = players[id];
      if (p && p.domElement && p.domElement.parentNode) p.domElement.parentNode.removeChild(p.domElement);
      delete players[id];
    }
    for (const id in statePlayers){
      createPlayerAvatar(statePlayers[id]);
    }
    // Update profile UI
    const pn = document.getElementById('profile-name');
    if (pn) pn.textContent = myName || 'You';
    const pa = document.getElementById('profile-avatar');
    if (pa) {
      pa.src = dicebearUrl(myName);
      pa.alt = myName || 'Your avatar';
    }
    updateAllAuras();
    updateStats();
  };

  window.handlePlayerJoined = (playerData) => {
    createPlayerAvatar(playerData);
  };

  window.handlePlayerLeft = (payload) => {
    const p = players[payload.id];
    if (!p) return;
    if (p.domElement && p.domElement.parentNode) p.domElement.parentNode.removeChild(p.domElement);
    delete players[payload.id];
  };

  window.handleBoopEvent = (payload) => {
    const { booperID, boopedID } = payload;
    // Remove opposite direction to keep only latest.
    if (boopLog[boopedID] && boopLog[boopedID][booperID]) {
      delete boopLog[boopedID][booperID];
      if (Object.keys(boopLog[boopedID]).length === 0) delete boopLog[boopedID];
    }
    if (!boopLog[booperID]) boopLog[booperID] = {};
    boopLog[booperID][boopedID] = true;
    if (boopedID === myID) playSound('sound-booped');
    else if (booperID === myID) playSound('sound-boop');
    // Update my stats counters
    if (booperID === myID) myBoopsMade++;
    if (boopedID === myID) myBoopsReceived++;
    updateAllAuras();
    updateStats();
    addBoopLog(booperID, boopedID);
  };

  window.handleBoopDenied = (payload) => {
    // Optionally flash the target avatar to indicate denial
    const { boopedID } = payload;
    const p = players[boopedID];
    if (p) {
      p.domElement.classList.add('deny-flash');
      setTimeout(() => p.domElement.classList.remove('deny-flash'), 300);
    }
  };
})();
