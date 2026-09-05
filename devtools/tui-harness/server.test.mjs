import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createServer } from "node:net";
import { setTimeout as delay } from "node:timers/promises";
import test from "node:test";

import { WebSocket } from "ws";

import { createTerminalEnvironment } from "./terminal-environment.mjs";

test("enables true color for the visual-debugging child process", () => {
  const environment = createTerminalEnvironment(
    { PATH: "test-path", TERM: "dumb", COLORTERM: "", NO_COLOR: "1" },
    "test-log-endpoint",
  );

  assert.equal(environment.PATH, "test-path");
  assert.equal(environment.TERM, "xterm-256color");
  assert.equal(environment.COLORTERM, "truecolor");
  assert.equal(environment.DOVIK_TUI_DEV_LOG_PIPE, "test-log-endpoint");
  assert.equal(environment.NO_COLOR, undefined);
});

async function reservePort() {
  const server = createServer();
  await new Promise((resolveListen, rejectListen) => {
    server.once("error", rejectListen);
    server.listen(0, "127.0.0.1", resolveListen);
  });
  const address = server.address();
  await new Promise((resolveClose) => server.close(resolveClose));
  return address.port;
}

function processExists(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

async function waitUntil(predicate, description, timeout = 10_000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const value = predicate();
    if (value) {
      return value;
    }
    await delay(25);
  }
  throw new Error(`Timed out waiting for ${description}`);
}

async function connect(url) {
  const messages = [];
  const socket = new WebSocket(url.replace("http:", "ws:") + "/terminal", {
    origin: url,
  });
  socket.on("message", (raw) => messages.push(JSON.parse(raw.toString())));
  await new Promise((resolveOpen, rejectOpen) => {
    socket.once("open", resolveOpen);
    socket.once("error", rejectOpen);
  });
  return { socket, messages };
}

test("runs the real TUI through ConPTY and cleans up its process", { timeout: 45_000 }, async (t) => {
  const port = await reservePort();
  const url = `http://127.0.0.1:${port}`;
  const child = spawn(process.execPath, ["server.mjs"], {
    cwd: import.meta.dirname,
    env: { ...process.env, DOVIK_TUI_HARNESS_PORT: String(port), DOVIK_TUI_HARNESS_TEST: "1" },
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true,
  });
  let output = "";
  child.stdout.setEncoding("utf8");
  child.stderr.setEncoding("utf8");
  child.stdout.on("data", (data) => {
    output += data;
  });
  child.stderr.on("data", (data) => {
    output += data;
  });
  t.after(() => {
    if (child.exitCode === null) {
      child.kill();
    }
  });

  await waitUntil(() => output.includes(`Dovik TUI harness: ${url}`), "harness startup", 20_000);
  const first = await connect(url);
  const started = await waitUntil(
    () => first.messages.find((message) => message.type === "status" && message.state === "running"),
    "first process startup",
  );
  assert.equal(processExists(started.pid), true);

  first.socket.send(JSON.stringify({ type: "resize", cols: 92, rows: 28 }));
  await waitUntil(
    () => first.messages.find((message) => message.type === "log" && message.entry.event === "terminal.resized" && message.entry.fields.cols === 92 && message.entry.fields.rows === 28),
    "PTY resize",
  );
  await waitUntil(
    () => first.messages.find((message) => message.type === "log" && message.entry.event === "terminal.size" && message.entry.fields.cols === 92 && message.entry.fields.rows === 28),
    "TUI resize observation",
  );
  await waitUntil(
    () => first.messages.find((message) => message.type === "log" && message.entry.event === "run.started"),
    "separate TUI diagnostics",
  );

  let daemonPid;
  if (process.platform === "win32") {
    await waitUntil(
      () => first.messages.find((message) => message.type === "log" && message.entry.event === "registry.failed"),
      "isolated registry failure",
    );
    first.socket.send(JSON.stringify({ type: "input", data: "s" }));
    await waitUntil(
      () => first.messages.find((message) => message.type === "log" && message.entry.event === "input.key" && message.entry.fields.key === "s"),
      "daemon start input",
    );
    const daemonStarted = await waitUntil(
      () => first.messages.find((message) => message.type === "log" && (message.entry.event === "daemon.launch.started" || message.entry.event === "daemon.launch.failed")),
      "detached daemon startup result",
    );
    assert.equal(daemonStarted.entry.event, "daemon.launch.started", JSON.stringify(daemonStarted.entry));
    daemonPid = daemonStarted.entry.fields.pid;
    await waitUntil(
      () => first.messages.find((message) => message.type === "log" && message.entry.event === "daemon.launch.ready"),
      "daemon protocol readiness",
    );
    assert.equal(processExists(daemonPid), true);
  }

  first.socket.send(JSON.stringify({ type: "binary", data: Buffer.from("?").toString("base64") }));
  await waitUntil(
    () => first.messages.find((message) => message.type === "log" && message.entry.event === "input.key" && message.entry.fields.key === "?"),
    "binary terminal input",
  );

  first.socket.send(JSON.stringify({ type: "input", data: "\x03" }));
  const exited = await waitUntil(
    () => first.messages.find((message) => message.type === "status" && message.state === "exited" && message.pid === started.pid),
    "Ctrl+C exit",
  );
  assert.equal(exited.exitCode, 0);
  if (daemonPid) {
    assert.equal(processExists(daemonPid), true);
  }

  const loadedBeforeRestart = first.messages.filter((message) => message.type === "log" && message.entry.event === "registry.loaded").length;
  first.socket.send(JSON.stringify({ type: "restart", cols: 88, rows: 26 }));
  const restarted = await waitUntil(
    () => first.messages.find((message) => message.type === "status" && message.state === "running" && message.pid !== started.pid),
    "process restart",
  );
  if (daemonPid) {
    await waitUntil(
      () => first.messages.filter((message) => message.type === "log" && message.entry.event === "registry.loaded").length > loadedBeforeRestart,
      "replacement TUI reconnect",
    );
    const daemonStarts = first.messages.filter((message) => message.type === "log" && message.entry.event === "daemon.launch.started");
    assert.equal(daemonStarts.length, 1);
    assert.equal(processExists(daemonPid), true);
  }
  first.socket.close();
  await waitUntil(() => !processExists(restarted.pid), "browser disconnect cleanup");
  if (daemonPid) {
    assert.equal(processExists(daemonPid), true);
  }

  const second = await connect(url);
  const finalProcess = await waitUntil(
    () => second.messages.find((message) => message.type === "status" && message.state === "running"),
    "final process startup",
  );
  second.socket.send(JSON.stringify({ type: "shutdownHarness" }));
  await new Promise((resolveExit, rejectExit) => {
    child.once("exit", resolveExit);
    child.once("error", rejectExit);
  });
  assert.equal(child.exitCode, 0, output);
  assert.equal(processExists(finalProcess.pid), false);
  if (daemonPid) {
    await waitUntil(() => !processExists(daemonPid), "fixture daemon cleanup");
  }
});
