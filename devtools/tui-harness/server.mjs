import { spawn, spawnSync } from "node:child_process";
import { createServer } from "node:http";
import { createServer as createPipeServer } from "node:net";
import { rmSync } from "node:fs";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import * as pty from "node-pty";
import { WebSocket, WebSocketServer } from "ws";

import { createTerminalEnvironment } from "./terminal-environment.mjs";

const harnessDirectory = dirname(fileURLToPath(import.meta.url));
const projectRoot = resolve(harnessDirectory, "..", "..");
const host = "127.0.0.1";
const port = Number.parseInt(process.env.DOVIK_TUI_HARNESS_PORT ?? "7331", 10);
const origin = `http://${host}:${port}`;
const temporaryDirectory = await mkdtemp(join(tmpdir(), "dovik-tui-harness-"));
const executable = join(temporaryDirectory, process.platform === "win32" ? "dovik.exe" : "dovik");
const logEndpoint = process.platform === "win32"
  ? `\\\\.\\pipe\\dovik-tui-harness-${process.pid}`
  : join(temporaryDirectory, "diagnostics.sock");

process.on("exit", () => {
  try {
    rmSync(temporaryDirectory, { recursive: true, force: true });
  } catch {
    // Best-effort fallback for startup failures; normal shutdown removes it first.
  }
});

if (!Number.isInteger(port) || port < 1 || port > 65535) {
  throw new Error("DOVIK_TUI_HARNESS_PORT must be a valid TCP port");
}

const build = spawnSync(
  "go",
  ["build", "-tags", "dovik_dev_harness", "-o", executable, "./cmd/dovik"],
  { cwd: projectRoot, env: process.env, stdio: "inherit", shell: false },
);
if (build.error) {
  await rm(temporaryDirectory, { recursive: true, force: true });
  throw build.error;
}
if (build.status !== 0) {
  await rm(temporaryDirectory, { recursive: true, force: true });
  process.exit(build.status ?? 1);
}

const sockets = new Set();
const sessions = new Map();
const logHistory = [];
const maxLogEntries = 500;
let shuttingDown = false;

function send(socket, message) {
  if (socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(message));
  }
}

function broadcast(message) {
  for (const socket of sockets) {
    send(socket, message);
  }
}

function appendLog(entry) {
  const normalized = {
    time: entry.time ?? new Date().toISOString(),
    source: entry.source ?? "harness",
    level: entry.level ?? "info",
    event: entry.event ?? "message",
    fields: entry.fields ?? {},
  };
  logHistory.push(normalized);
  if (logHistory.length > maxLogEntries) {
    logHistory.splice(0, logHistory.length - maxLogEntries);
  }
  broadcast({ type: "log", entry: normalized });
}

async function terminateProcessTree(pid) {
  if (!pid) {
    return;
  }
  if (process.platform === "win32") {
    await new Promise((done) => {
      const killer = spawn("taskkill.exe", ["/PID", String(pid), "/T", "/F"], {
        windowsHide: true,
        stdio: "ignore",
      });
      killer.once("error", done);
      killer.once("exit", done);
    });
    return;
  }
  try {
    process.kill(-pid, "SIGTERM");
  } catch {
    try {
      process.kill(pid, "SIGTERM");
    } catch {
      // The process already exited.
    }
  }
}

function delay(milliseconds) {
  return new Promise((resolveDelay) => setTimeout(resolveDelay, milliseconds));
}

class TerminalSession {
  constructor(socket) {
    this.socket = socket;
    this.process = undefined;
    this.running = false;
    this.exitPromise = Promise.resolve();
    this.stopPromise = undefined;
  }

  start(cols = 100, rows = 30) {
    if (this.running) {
      return;
    }
    cols = Number.isInteger(cols) && cols >= 2 && cols <= 1000 ? cols : 100;
    rows = Number.isInteger(rows) && rows >= 1 && rows <= 500 ? rows : 30;
    const environment = createTerminalEnvironment(process.env, logEndpoint);
    const terminalProcess = pty.spawn(executable, ["tui"], {
      name: "xterm-256color",
      cols,
      rows,
      cwd: projectRoot,
      env: environment,
      useConpty: process.platform === "win32",
    });
    this.process = terminalProcess;
    this.running = true;
    this.stopPromise = undefined;
    let resolveExit;
    this.exitPromise = new Promise((resolveProcessExit) => {
      resolveExit = resolveProcessExit;
    });

    appendLog({ event: "process.started", fields: { pid: terminalProcess.pid, cols, rows } });
    send(this.socket, { type: "status", state: "running", pid: terminalProcess.pid });

    terminalProcess.onData((data) => send(this.socket, { type: "output", data }));
    terminalProcess.onExit(({ exitCode, signal }) => {
      this.running = false;
      appendLog({ event: "process.exited", fields: { pid: terminalProcess.pid, exitCode, signal } });
      send(this.socket, { type: "status", state: "exited", pid: terminalProcess.pid, exitCode, signal });
      resolveExit();
    });
  }

  write(data) {
    if (this.running && (typeof data === "string" || Buffer.isBuffer(data))) {
      this.process.write(data);
    }
  }

  resize(cols, rows) {
    if (!this.running || !Number.isInteger(cols) || !Number.isInteger(rows)) {
      return;
    }
    if (cols < 2 || rows < 1 || cols > 1000 || rows > 500) {
      return;
    }
    this.process.resize(cols, rows);
    appendLog({ event: "terminal.resized", fields: { cols, rows } });
  }

  stop(reason) {
    if (!this.running) {
      return Promise.resolve();
    }
    if (!this.stopPromise) {
      this.stopPromise = this.stopProcess(reason).finally(() => {
        this.stopPromise = undefined;
      });
    }
    return this.stopPromise;
  }

  async stopProcess(reason) {
    const pid = this.process.pid;
    appendLog({ event: "process.stopping", fields: { pid, reason } });
    this.process.write("\x03");
    await Promise.race([this.exitPromise, delay(500)]);
    if (this.running) {
      await terminateProcessTree(pid);
      await Promise.race([this.exitPromise, delay(1500)]);
    }
    if (this.running) {
      try {
        this.process.kill();
      } catch {
        // The PTY closed between the state check and the fallback.
      }
    }
  }
}

const diagnosticsServer = createPipeServer((connection) => {
  connection.setEncoding("utf8");
  let buffered = "";
  connection.on("data", (data) => {
    buffered += data;
    for (;;) {
      const newline = buffered.indexOf("\n");
      if (newline < 0) {
        break;
      }
      const line = buffered.slice(0, newline);
      buffered = buffered.slice(newline + 1);
      if (!line) {
        continue;
      }
      try {
        appendLog(JSON.parse(line));
      } catch {
        appendLog({ level: "warn", event: "diagnostics.invalid_record" });
      }
    }
  });
});

await new Promise((resolveListen, rejectListen) => {
  diagnosticsServer.once("error", rejectListen);
  diagnosticsServer.listen(logEndpoint, resolveListen);
});

const contentTypes = new Map([
  [".css", "text/css; charset=utf-8"],
  [".html", "text/html; charset=utf-8"],
  [".js", "text/javascript; charset=utf-8"],
  [".mjs", "text/javascript; charset=utf-8"],
]);

const routes = new Map([
  ["/", join(harnessDirectory, "public", "index.html")],
  ["/app.js", join(harnessDirectory, "public", "app.js")],
  ["/styles.css", join(harnessDirectory, "public", "styles.css")],
  ["/vendor/xterm.mjs", join(harnessDirectory, "node_modules", "@xterm", "xterm", "lib", "xterm.mjs")],
  ["/vendor/xterm.css", join(harnessDirectory, "node_modules", "@xterm", "xterm", "css", "xterm.css")],
  ["/vendor/addon-fit.mjs", join(harnessDirectory, "node_modules", "@xterm", "addon-fit", "lib", "addon-fit.mjs")],
]);

const httpServer = createServer(async (request, response) => {
  const requestUrl = new URL(request.url ?? "/", origin);
  const path = routes.get(requestUrl.pathname);
  if (request.method !== "GET" || !path) {
    response.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
    response.end("Not found\n");
    return;
  }
  try {
    const content = await readFile(path);
    response.writeHead(200, {
      "Cache-Control": "no-store",
      "Content-Security-Policy": "default-src 'self'; connect-src 'self' ws://127.0.0.1:*; style-src 'self' 'unsafe-inline'; img-src 'none'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'",
      "Content-Type": contentTypes.get(extname(path)) ?? "application/octet-stream",
      "X-Content-Type-Options": "nosniff",
    });
    response.end(content);
  } catch (error) {
    response.writeHead(500, { "Content-Type": "text/plain; charset=utf-8" });
    response.end(`Failed to read asset: ${error.message}\n`);
  }
});

const webSocketServer = new WebSocketServer({ noServer: true });
httpServer.on("upgrade", (request, socket, head) => {
  if (request.url !== "/terminal" || request.headers.origin !== origin) {
    socket.destroy();
    return;
  }
  webSocketServer.handleUpgrade(request, socket, head, (webSocket) => {
    webSocketServer.emit("connection", webSocket, request);
  });
});

webSocketServer.on("connection", (socket) => {
  sockets.add(socket);
  const session = new TerminalSession(socket);
  sessions.set(socket, session);
  send(socket, { type: "logs", entries: logHistory });
  session.start();

  socket.on("message", async (raw) => {
    let message;
    try {
      message = JSON.parse(raw.toString());
    } catch {
      return;
    }
    if (message.type === "input") {
      session.write(message.data);
    } else if (message.type === "binary" && typeof message.data === "string") {
      session.write(Buffer.from(message.data, "base64"));
    } else if (message.type === "resize") {
      session.resize(message.cols, message.rows);
    } else if (message.type === "restart") {
      await session.stop("restart");
      session.start(message.cols, message.rows);
    } else if (message.type === "clearLogs") {
      logHistory.length = 0;
      broadcast({ type: "logsCleared" });
    } else if (message.type === "shutdownHarness" && process.env.DOVIK_TUI_HARNESS_TEST === "1") {
      await shutdown("test request");
    }
  });

  socket.once("close", async () => {
    sockets.delete(socket);
    sessions.delete(socket);
    await session.stop("browser disconnected");
  });
});

await new Promise((resolveListen, rejectListen) => {
  httpServer.once("error", rejectListen);
  httpServer.listen(port, host, resolveListen);
});

console.log(`Dovik TUI harness: ${origin}`);

async function shutdown(signal) {
  if (shuttingDown) {
    return;
  }
  shuttingDown = true;
  appendLog({ event: "server.stopping", fields: { signal } });
  const activeSessions = [...sessions.values()];
  for (const socket of sockets) {
    socket.close(1001, "Harness stopping");
  }
  await Promise.all(activeSessions.map((session) => session.stop("server shutdown")));
  await Promise.all([
    new Promise((done) => webSocketServer.close(done)),
    new Promise((done) => httpServer.close(done)),
    new Promise((done) => diagnosticsServer.close(done)),
  ]);
  await rm(temporaryDirectory, { recursive: true, force: true });
  process.exit(0);
}

process.on("SIGINT", () => void shutdown("SIGINT"));
process.on("SIGTERM", () => void shutdown("SIGTERM"));
