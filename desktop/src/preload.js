const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("linkbit", {
  loadSettings: () => ipcRenderer.invoke("settings:load"),
  startAgent: (settings) => ipcRenderer.invoke("agent:start", settings),
  stopAgent: () => ipcRenderer.invoke("agent:stop"),
  startForward: (settings) => ipcRenderer.invoke("forward:start", settings),
  stopForward: (listen) => ipcRenderer.invoke("forward:stop", listen),
  openConsole: (controller) => ipcRenderer.invoke("console:open", controller),
  onStatus: (callback) => ipcRenderer.on("agent:status", (_event, value) => callback(value)),
  onLog: (callback) => ipcRenderer.on("agent:log", (_event, value) => callback(value))
});
