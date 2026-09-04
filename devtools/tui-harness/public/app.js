import { Terminal } from "/vendor/xterm.mjs";
import { FitAddon } from "/vendor/addon-fit.mjs";

const terminalElement = document.querySelector("[data-testid='terminal']");
const statusElement = document.querySelector("[data-testid='process-status']");
const statusDot = document.querySelector("[data-testid='status-dot']");
const pidElement = document.querySelector("[data-testid='process-pid']");
const restartButton = document.querySelector("[data-testid='restart']");
const clearLogsButton = document.querySelector("[data-testid='clear-logs']");
const logOutput = document.querySelector("[data-testid='log-output']");

const terminal = new Terminal({
  allowProposedApi: false,
  allowTransparency: false,
  convertEol: false,
  cursorBlink: false,
  cursorStyle: "block",
  fontFamily: '"Cascadia Mono", "SFMono-Regular", Consolas, "Liberation Mono", monospace',
  fontSize: 14,
  fontWeight: "400",
  lineHeight: 1.15,
  scrollback: 0,
  theme: {
    background: "#111318",
    foreground: "#e6e9ef",
    cursor: "#e6e9ef",
    cursorAccent: "#111318",
    selectionBackground: "#425268",
    black: "#171a21",
    red: "#ef6a6a",
    green: "#7fcf92",
    yellow: "#e6c06a",
    blue: "#78a9ff",
    magenta: "#c99afc",
    cyan: "#63cdda",
    white: "#e6e9ef",
    brightBlack: "#727987",
    brightRed: "#ff8484",
    brightGreen: "#98e6aa",
    brightYellow: "#f3d27f",
    brightBlue: "#91bbff",
    brightMagenta: "#d8afff",
    brightCyan: "#80e3ea",
    brightWhite: "#ffffff",
  },
});
const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);
terminal.open(terminalElement);

const socketProtocol = location.protocol === "https:" ? "wss:" : "ws:";
const socket = new WebSocket(`${socketProtocol}//${location.host}/terminal`);
let resizeTimer;

function send(message) {
  if (socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(message));
  }
}

function fitTerminal() {
  fitAddon.fit();
  send({ type: "resize", cols: terminal.cols, rows: terminal.rows });
}

function setStatus(state, pid, exitCode) {
  statusDot.dataset.state = state;
  if (state === "running") {
    statusElement.textContent = "Running";
    pidElement.textContent = `PID ${pid}`;
    restartButton.disabled = false;
  } else if (state === "exited") {
    statusElement.textContent = `Exited (${exitCode ?? "unknown"})`;
    pidElement.textContent = pid ? `PID ${pid}` : "";
    restartButton.disabled = false;
  } else {
    statusElement.textContent = state;
    pidElement.textContent = "";
    restartButton.disabled = true;
  }
}

function formatLog(entry) {
  const time = new Date(entry.time).toLocaleTimeString("en-GB", { hour12: false });
  const fields = Object.entries(entry.fields ?? {})
    .map(([key, value]) => `${key}=${JSON.stringify(value)}`)
    .join(" ");
  return `${time}  ${entry.source.padEnd(7)}  ${entry.event}${fields ? `  ${fields}` : ""}`;
}

function appendLog(entry) {
  const row = document.createElement("div");
  row.className = `log-row level-${entry.level}`;
  row.textContent = formatLog(entry);
  logOutput.append(row);
  while (logOutput.childElementCount > 500) {
    logOutput.firstElementChild.remove();
  }
  logOutput.scrollTop = logOutput.scrollHeight;
}

socket.addEventListener("open", () => {
  setStatus("Connected");
  requestAnimationFrame(() => {
    fitTerminal();
    terminal.focus();
  });
});

socket.addEventListener("message", (event) => {
  const message = JSON.parse(event.data);
  if (message.type === "output") {
    terminal.write(message.data);
  } else if (message.type === "status") {
    setStatus(message.state, message.pid, message.exitCode);
  } else if (message.type === "log") {
    appendLog(message.entry);
  } else if (message.type === "logs") {
    logOutput.replaceChildren();
    for (const entry of message.entries) {
      appendLog(entry);
    }
  } else if (message.type === "logsCleared") {
    logOutput.replaceChildren();
  }
});

socket.addEventListener("close", () => setStatus("Disconnected"));
socket.addEventListener("error", () => setStatus("Connection error"));

terminal.onData((data) => send({ type: "input", data }));
terminal.onBinary((data) => {
  const bytes = Uint8Array.from(data, (character) => character.charCodeAt(0));
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  send({ type: "binary", data: btoa(binary) });
});

const resizeObserver = new ResizeObserver(() => {
  clearTimeout(resizeTimer);
  resizeTimer = setTimeout(fitTerminal, 80);
});
resizeObserver.observe(terminalElement);

restartButton.addEventListener("click", () => {
  terminal.reset();
  setStatus("Restarting");
  send({ type: "restart", cols: terminal.cols, rows: terminal.rows });
  terminal.focus();
});

clearLogsButton.addEventListener("click", () => send({ type: "clearLogs" }));
terminalElement.addEventListener("click", () => terminal.focus());
window.addEventListener("beforeunload", () => socket.close());
