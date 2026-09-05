import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

// Exercise the browser event wiring without adding a DOM implementation dependency.
async function loadApp() {
  class Element {
    children = [];
    handlers = {};
    attributes = {};
    dataset = {};
    hidden = false;
    classList = { toggle() {} };
    addEventListener(name, callback) { this.handlers[name] = callback; }
    setAttribute(name, value) { this.attributes[name] = value; }
    getAttribute(name) { return this.attributes[name]; }
    append(row) { row.remove = () => this.children.shift(); this.children.push(row); }
    replaceChildren() { this.children = []; }
    get childElementCount() { return this.children.length; }
    get firstElementChild() { return this.children[0]; }
    get scrollHeight() { return this.children.length; }
  }
  const elements = new Map();
  const element = (selector) => {
    if (!elements.has(selector)) elements.set(selector, new Element());
    return elements.get(selector);
  };
  const toggle = element("[data-testid='diagnostics-toggle']");
  toggle.setAttribute("aria-expanded", "false");
  element("#diagnostics").hidden = true;
  const sent = [];
  let socket;
  let fits = 0;
  let resets = 0;
  let focuses = 0;
  class Socket extends Element {
    static OPEN = 1;
    readyState = 1;
    constructor() { super(); socket = this; }
    send(message) { sent.push(JSON.parse(message)); }
  }
  const source = (await readFile(new URL("./public/app.js", import.meta.url), "utf8"))
    .replace(/^import .*;\r?\n/gm, "");
  runInNewContext(source, {
    document: { querySelector: element, createElement: () => new Element() },
    location: { protocol: "http:", host: "127.0.0.1:7331" },
    WebSocket: Socket,
    Terminal: class {
      cols = 100; rows = 30;
      loadAddon() {} open() {} onData() {} onBinary() {}
      reset() { resets++; } focus() { focuses++; }
    },
    FitAddon: class { fit() { fits++; } },
    ResizeObserver: class { observe() {} },
    requestAnimationFrame: (callback) => callback(),
    setTimeout, clearTimeout,
    window: { addEventListener() {} },
  });
  return { element, toggle, sent, message: (data) => socket.handlers.message({ data: JSON.stringify(data) }),
    counts: () => ({ fits, resets, focuses }) };
}

test("diagnostic disclosure retains bounded live history and only resizes the PTY", async () => {
  const app = await loadApp();
  const panel = app.element("#diagnostics");
  const output = app.element("[data-testid='log-output']");
  assert.equal(panel.hidden, true);
  for (let index = 0; index < 510; index++) {
    app.message({ type: "log", entry: { time: 0, source: "tui", event: `event-${index}` } });
  }
  assert.equal(output.childElementCount, 500);
  assert.match(output.children[0].textContent, /event-10$/);
  app.toggle.handlers.click();
  assert.equal(panel.hidden, false);
  assert.equal(app.toggle.getAttribute("aria-expanded"), "true");
  assert.equal(output.scrollTop, 500);
  app.toggle.handlers.click();
  assert.equal(panel.hidden, true);
  assert.equal(output.childElementCount, 500);
  assert.deepEqual(app.sent.map((message) => message.type), ["resize", "resize"]);
  assert.deepEqual(app.counts(), { fits: 2, resets: 0, focuses: 0 });
  app.element("[data-testid='clear-logs']").handlers.click();
  assert.equal(app.sent.at(-1).type, "clearLogs");
  app.message({ type: "logsCleared" });
  assert.equal(output.childElementCount, 0);
});

test("status and restart refer to the TUI child", async () => {
  const app = await loadApp();
  app.message({ type: "status", state: "running", pid: 42 });
  assert.equal(app.element("[data-testid='process-status']").textContent, "Running");
  assert.equal(app.element("[data-testid='process-pid']").textContent, "TUI PID 42");
  app.element("[data-testid='restart']").handlers.click();
  assert.equal(app.sent.at(-1).type, "restart");
  assert.equal(app.element("[data-testid='process-status']").textContent, "Restarting");
  app.message({ type: "status", state: "exited", pid: 42, exitCode: 0 });
  assert.equal(app.element("[data-testid='process-status']").textContent, "Exited (0)");
});
