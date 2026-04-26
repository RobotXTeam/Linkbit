const { app, BrowserWindow, ipcMain, shell } = require("electron");
const Store = require("electron-store");
const { spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const store = new Store({ name: "linkbit-client" });
let agentProcess = null;
const forwardProcesses = new Map();

app.setName("Linkbit");

function agentBinary() {
  const ext = process.platform === "win32" ? ".exe" : "";
  const candidates = [
    path.join(process.resourcesPath || "", "bin", `linkbit-agent${ext}`),
    path.join(app.getAppPath(), "..", "bin", `linkbit-agent${ext}`),
    path.join(process.cwd(), "bin", `linkbit-agent${ext}`),
    `linkbit-agent${ext}`
  ];
  return candidates.find((candidate) => candidate === `linkbit-agent${ext}` || fs.existsSync(candidate));
}

function appIcon() {
  const candidates = [
    path.join(__dirname, "..", "build", "icon.png"),
    path.join(process.resourcesPath || "", "build", "icon.png"),
    path.join(process.resourcesPath || "", "icon.png")
  ];
  return candidates.find((candidate) => fs.existsSync(candidate));
}

function normalizeControllerURL(input) {
  const trimmed = String(input || "").trim();
  if (!trimmed) {
    return "";
  }
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(trimmed) ? trimmed : `http://${trimmed}`;
  try {
    const url = new URL(withScheme);
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      return "";
    }
    return url.toString();
  } catch {
    return "";
  }
}

function createWindow() {
  const win = new BrowserWindow({
    width: 980,
    height: 720,
    minWidth: 820,
    minHeight: 620,
    title: "Linkbit",
    icon: appIcon(),
    webPreferences: {
      preload: path.join(__dirname, "preload.js")
    }
  });
  win.loadFile(path.join(__dirname, "renderer.html"));
}

ipcMain.handle("settings:load", () => ({
  controller: store.get("controller", ""),
  enrollmentKey: store.get("enrollmentKey", ""),
  deviceName: store.get("deviceName", os.hostname()),
  interfaceName: store.get("interfaceName", "linkbit0"),
  statePath: store.get("statePath", path.join(app.getPath("userData"), "agent-state.json")),
  dryRun: store.get("dryRun", false),
  forwardListen: store.get("forwardListen", "127.0.0.1:10022"),
  forwardTarget: store.get("forwardTarget", "")
}));

ipcMain.handle("agent:start", (_event, input) => {
  if (agentProcess) {
    return { ok: false, message: "Agent is already running" };
  }
  const settings = {
    controller: String(input.controller || "").trim(),
    enrollmentKey: String(input.enrollmentKey || "").trim(),
    deviceName: String(input.deviceName || os.hostname()).trim(),
    interfaceName: String(input.interfaceName || "linkbit0").trim(),
    statePath: String(input.statePath || path.join(app.getPath("userData"), "agent-state.json")).trim(),
    dryRun: Boolean(input.dryRun)
  };
  if (!settings.controller) {
    return { ok: false, message: "Controller URL is required" };
  }
  store.set(settings);
  fs.mkdirSync(path.dirname(settings.statePath), { recursive: true });

  const args = [
    "--controller", settings.controller,
    "--name", settings.deviceName,
    "--state", settings.statePath,
    "--interface", settings.interfaceName
  ];
  if (settings.enrollmentKey) {
    args.push("--enrollment-key", settings.enrollmentKey);
  }
  if (settings.dryRun) {
    args.push("--dry-run");
  }

  agentProcess = spawn(agentBinary(), args, {
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true
  });
  agentProcess.on("exit", (code) => {
    agentProcess = null;
    BrowserWindow.getAllWindows().forEach((win) => win.webContents.send("agent:status", { running: false, code }));
  });
  agentProcess.stdout.on("data", (chunk) => {
    BrowserWindow.getAllWindows().forEach((win) => win.webContents.send("agent:log", chunk.toString()));
  });
  agentProcess.stderr.on("data", (chunk) => {
    BrowserWindow.getAllWindows().forEach((win) => win.webContents.send("agent:log", chunk.toString()));
  });
  return { ok: true, message: "Agent started" };
});

ipcMain.handle("agent:stop", () => {
  if (!agentProcess) {
    return { ok: true, message: "Agent is not running" };
  }
  agentProcess.kill();
  agentProcess = null;
  return { ok: true, message: "Agent stopped" };
});

ipcMain.handle("forward:start", (_event, input) => {
  const controller = String(input.controller || "").trim();
  const statePath = String(input.statePath || path.join(app.getPath("userData"), "agent-state.json")).trim();
  const listen = String(input.listen || "127.0.0.1:10022").trim();
  const target = String(input.target || "").trim();
  if (!controller || !statePath || !listen || !target) {
    return { ok: false, message: "Controller, state file, listen address, and target are required" };
  }
  if (forwardProcesses.has(listen)) {
    return { ok: false, message: `Forward is already running on ${listen}` };
  }
  store.set({ forwardListen: listen, forwardTarget: target });
  const child = spawn(agentBinary(), [
    "forward",
    "--controller", controller,
    "--state", statePath,
    "--listen", listen,
    "--target", target
  ], {
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true
  });
  forwardProcesses.set(listen, child);
  child.on("exit", (code) => {
    forwardProcesses.delete(listen);
    BrowserWindow.getAllWindows().forEach((win) => win.webContents.send("agent:status", { running: Boolean(agentProcess), code }));
  });
  child.stdout.on("data", (chunk) => {
    BrowserWindow.getAllWindows().forEach((win) => win.webContents.send("agent:log", chunk.toString()));
  });
  child.stderr.on("data", (chunk) => {
    BrowserWindow.getAllWindows().forEach((win) => win.webContents.send("agent:log", chunk.toString()));
  });
  return { ok: true, message: `Forward started on ${listen}` };
});

ipcMain.handle("forward:stop", (_event, listen) => {
  const key = String(listen || "").trim();
  const child = forwardProcesses.get(key);
  if (!child) {
    return { ok: true, message: "Forward is not running" };
  }
  child.kill();
  forwardProcesses.delete(key);
  return { ok: true, message: `Forward stopped on ${key}` };
});

ipcMain.handle("console:open", async (_event, controller) => {
  const target = normalizeControllerURL(controller);
  if (!target) {
    return { ok: false, message: "Controller URL is required" };
  }
  try {
    await shell.openExternal(target);
  } catch (error) {
    return { ok: false, message: `Open console failed: ${error.message || error}` };
  }
  return { ok: true, message: `Opened ${target}` };
});

app.whenReady().then(createWindow);
app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});
app.on("before-quit", () => {
  if (agentProcess) {
    agentProcess.kill();
  }
  for (const child of forwardProcesses.values()) {
    child.kill();
  }
});
