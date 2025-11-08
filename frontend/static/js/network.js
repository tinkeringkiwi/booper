// network.js - handles websocket connection and dispatches to main.js (classic script, exposes window.network)
(function(){
  const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
  const url = protocol + window.location.host + '/ws';
  let socket = null;
  let reconnectAttempts = 0;
  const maxDelay = 8000;

  function connect(){
    socket = new WebSocket(url);
    socket.onopen = () => {
      reconnectAttempts = 0;
      console.log('[net] connected');
    };
    socket.onclose = () => {
      console.log('[net] disconnected');
      scheduleReconnect();
    };
    socket.onerror = (e) => {
      console.error('[net] error', e);
      // error often followed by close
    };
    socket.onmessage = (ev) => {
      let msg;
      try { msg = JSON.parse(ev.data); } catch (e) { console.warn('bad json', e); return; }
      switch (msg.type) {
        case 'welcome': window.handleWelcome && window.handleWelcome(msg.payload); break;
        case 'player_joined': window.handlePlayerJoined && window.handlePlayerJoined(msg.payload); break;
        case 'player_left': window.handlePlayerLeft && window.handlePlayerLeft(msg.payload); break;
        case 'boop_event': window.handleBoopEvent && window.handleBoopEvent(msg.payload); break;
        case 'boop_denied': window.handleBoopDenied && window.handleBoopDenied(msg.payload); break;
      }
    };
  }

  function scheduleReconnect(){
    const delay = Math.min(maxDelay, 500 * Math.pow(2, reconnectAttempts++));
    setTimeout(() => {
      console.log('[net] reconnecting…');
      connect();
    }, delay);
  }

  function sendBoop(boopedID){
    if (!boopedID || !socket || socket.readyState !== WebSocket.OPEN) {
      window.handleBoopDenied && window.handleBoopDenied({ boopedID });
      return;
    }
    try {
      socket.send(JSON.stringify({ type: 'boop_request', payload: { boopedID } }));
    } catch (e) {
      console.warn('sendBoop failed', e);
    }
  }

  connect();
  window.network = { sendBoop };
})();
