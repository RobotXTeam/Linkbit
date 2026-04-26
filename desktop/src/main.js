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

function defaultStatePath() {
  return path.join(app.getPath("userData"), "agent-state.json");
}

function inspectStateFile(statePath) {
  if (!statePath) {
    return { ok: false, message: "State file path is required" };
  }
  if (!fs.existsSync(statePath)) {
    return { ok: false, missing: true, message: `State file not found: ${statePath}` };
  }
  try {
    fs.accessSync(statePath, fs.constants.R_OK);
  } catch {
    return {
      ok: false,
      unreadable: true,
      message: `State file is not readable by this desktop app: ${statePath}`
    };
  }
  try {
    const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
    const hasDeviceCredentials = Boolean(state?.device?.id && state?.device?.deviceToken);
    const hasWireGuardIdentity = Boolean(state?.wireGuardPrivateKey && state?.wireGuardPublicKey);
    return { ok: hasDeviceCredentials, hasDeviceCredentials, hasWireGuardIdentity };
  } catch (error) {
    return { ok: false, message: `State file is invalid JSON: ${error.message || error}` };
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
  statePath: store.get("statePath", defaultStatePath()),
  dryRun: store.get("dryRun", false),
  forwardListen: store.get("forwardListen", "127.0.0.1:10022"),
  forwardTarget: store.get("forwardTarget", "")
}));

ipcMain.handle("agent:start", (_event, input) => {
  if (agentProcess) {
    return { ok: false, message: "Agent is already running" };
  }
  const settings = {
    controller: normalizeControllerURL(input.controller),
    enrollmentKey: String(input.enrollmentKey || "").trim(),
    deviceName: String(input.deviceName || os.hostname()).trim(),
    interfaceName: String(input.interfaceName || "linkbit0").trim(),
    statePath: String(input.statePath || defaultStatePath()).trim(),
    dryRun: Boolean(input.dryRun)
  };
  if (!settings.controller) {
    return { ok: false, message: "Controller URL is required, for example https://controller.example.com" };
  }
  const stateStatus = inspectStateFile(settings.statePath);
  if (!settings.enrollmentKey && (!stateStatus.ok || !stateStatus.hasDeviceCredentials)) {
    if (stateStatus.missing) {
      return { ok: false, message: "Enrollment key is required for first registration" };
    }
    if (stateStatus.unreadable) {
      return {
        ok: false,
        message: `${stateStatus.message}. Copy the system agent state into this user profile or choose a readable state file.`
      };
    }
    return {
      ok: false,
      message: "Agent state is missing device credentials. Enter an enrollment key or choose a state file that contains a registered device."
    };
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
  const controller = normalizeControllerURL(input.controller);
  const statePath = String(input.statePath || defaultStatePath()).trim();
  const listen = String(input.listen || "127.0.0.1:10022").trim();
  const target = String(input.target || "").trim();
  if (!controller) {
    return { ok: false, message: "Controller URL is required, for example https://controller.example.com" };
  }
  if (!statePath || !listen || !target) {
    return { ok: false, message: "State file, listen address, and target are required" };
  }
  const stateStatus = inspectStateFile(statePath);
  if (!stateStatus.ok || !stateStatus.hasDeviceCredentials) {
    if (stateStatus.missing) {
      return { ok: false, message: `${stateStatus.message}. Start Agent with an enrollment key first.` };
    }
    if (stateStatus.unreadable) {
      return {
        ok: false,
        message: `${stateStatus.message}. Copy /var/lib/linkbit/agent-state.json to this user's Linkbit state file or run the forwarder with permissions that can read it.`
      };
    }
    return {
      ok: false,
      message: "State file is missing device credentials. Use a registered Agent state file before starting SSH/RDP forwarding."
    };
  }
  if (forwardProcesses.has(listen)) {
    return { ok: false, message: `Forward is already running on ${listen}` };
  }
  store.set({ controller, statePath, forwardListen: listen, forwardTarget: target });
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
