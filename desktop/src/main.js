const { app, BrowserWindow, ipcMain, shell } = require("electron");
const Store = require("electron-store");
const { spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const store = new Store({ name: "linkbit-client" });
let agentProcess = null;

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

function createWindow() {
  const win = new BrowserWindow({
    width: 980,
    height: 720,
    minWidth: 820,
    minHeight: 620,
    title: "Linkbit",
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
  dryRun: store.get("dryRun", false)
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

ipcMain.handle("console:open", (_event, controller) => {
  if (controller) {
    shell.openExternal(controller);
  }
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
});
