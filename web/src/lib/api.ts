import { z } from "zod";

const API_KEY_STORAGE = "linkbit.adminApiKey";

const overviewSchema = z.object({
  onlineDevices: z.number(),
  totalDevices: z.number(),
  relayNodes: z.number(),
  healthyRelays: z.number(),
  policyCount: z.number(),
  networkHealth: z.string(),
  averageLoad: z.number()
});

const settingsSchema = z.object({
  publicUrl: z.string(),
  listenAddr: z.string(),
  logLevel: z.string(),
  webConsoleEnabled: z.boolean(),
  databaseBackend: z.string()
});

const deviceSchema = z.object({
  id: z.string(),
  userId: z.string(),
  groupId: z.string(),
  name: z.string(),
  virtualIp: z.string(),
  publicKey: z.string(),
  endpoint: z.string().optional(),
  status: z.string(),
  lastSeenAt: z.string(),
  createdAt: z.string(),
  fingerprint: z.string()
});

const userSchema = z.object({
  id: z.string(),
  name: z.string(),
  email: z.string().optional(),
  role: z.string(),
  createdAt: z.string()
});

const groupSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  createdAt: z.string()
});

const relaySchema = z.object({
  id: z.string(),
  name: z.string(),
  region: z.string(),
  publicUrl: z.string(),
  status: z.string(),
  load: z.number(),
  lastSeenAt: z.string()
});

const derpMapSchema = z.object({
  Regions: z.record(z.string(), z.unknown()).optional(),
  omitDefaultRegions: z.boolean().optional()
});

const apiKeySchema = z.object({
  id: z.string(),
  name: z.string(),
  scope: z.string(),
  createdAt: z.string(),
  lastUsedAt: z.string().optional(),
  revokedAt: z.string().optional(),
  key: z.string().optional()
});

const policySchema = z.object({
  id: z.string(),
  name: z.string(),
  sourceId: z.string(),
  targetId: z.string(),
  ports: z.array(z.string()),
  protocol: z.string(),
  enabled: z.boolean()
});

const invitationSchema = z.object({
  id: z.string(),
  userId: z.string(),
  groupId: z.string(),
  reusable: z.boolean(),
  expiresAt: z.string(),
  createdAt: z.string(),
  token: z.string().optional()
});

export type Overview = z.infer<typeof overviewSchema>;
export type ControllerSettings = z.infer<typeof settingsSchema>;
export type Device = z.infer<typeof deviceSchema>;
export type User = z.infer<typeof userSchema>;
export type DeviceGroup = z.infer<typeof groupSchema>;
export type RelayNode = z.infer<typeof relaySchema>;
export type NetworkPolicy = z.infer<typeof policySchema>;
export type Invitation = z.infer<typeof invitationSchema>;
export type APIKey = z.infer<typeof apiKeySchema>;

export function getStoredAPIKey() {
  return window.localStorage.getItem(API_KEY_STORAGE) ?? "";
}

export function storeAPIKey(value: string) {
  window.localStorage.setItem(API_KEY_STORAGE, value.trim());
}

export async function getOverview(apiKey: string) {
  return overviewSchema.parse(await request("/api/v1/overview", apiKey));
}

export async function getSettings(apiKey: string) {
  return settingsSchema.parse(await request("/api/v1/settings", apiKey));
}

export async function getDevices(apiKey: string) {
  return z.array(deviceSchema).parse(await request("/api/v1/devices", apiKey));
}

export async function deleteDevice(apiKey: string, id: string) {
  await request(`/api/v1/devices/${encodeURIComponent(id)}`, apiKey, { method: "DELETE" });
}

export async function getUsers(apiKey: string) {
  return z.array(userSchema).parse(await request("/api/v1/users", apiKey));
}

export async function createUser(apiKey: string, input: { id: string; name: string; email: string; role: string }) {
  return userSchema.parse(
    await request("/api/v1/users", apiKey, {
      method: "POST",
      body: JSON.stringify(input)
    })
  );
}

export async function getGroups(apiKey: string) {
  return z.array(groupSchema).parse(await request("/api/v1/groups", apiKey));
}

export async function createGroup(apiKey: string, input: { id: string; name: string; description: string }) {
  return groupSchema.parse(
    await request("/api/v1/groups", apiKey, {
      method: "POST",
      body: JSON.stringify(input)
    })
  );
}

export async function getRelays(apiKey: string) {
  return z.array(relaySchema).parse(await request("/api/v1/relays", apiKey));
}

export async function getDERPMap(apiKey: string) {
  return derpMapSchema.parse(await request("/api/v1/derp-map", apiKey));
}

export async function registerRelay(
  apiKey: string,
  input: { id: string; name: string; region: string; publicUrl: string }
) {
  return relaySchema.parse(
    await request("/api/v1/relays/register", apiKey, {
      method: "POST",
      body: JSON.stringify(input)
    })
  );
}

export async function deleteRelay(apiKey: string, id: string) {
  await request(`/api/v1/relays/${encodeURIComponent(id)}`, apiKey, { method: "DELETE" });
}

export async function getPolicies(apiKey: string) {
  return z.array(policySchema).parse(await request("/api/v1/policies", apiKey));
}

export async function getAPIKeys(apiKey: string) {
  return z.array(apiKeySchema).parse(await request("/api/v1/api-keys", apiKey));
}

export async function revokeAPIKey(apiKey: string, id: string) {
  await request(`/api/v1/api-keys/${encodeURIComponent(id)}`, apiKey, { method: "DELETE" });
}

export async function createAPIKey(apiKey: string, scope: "admin" | "relay") {
  return apiKeySchema.parse(
    await request("/api/v1/api-keys", apiKey, {
      method: "POST",
      body: JSON.stringify({
        name: `${scope}-${new Date().toISOString()}`,
        scope
      })
    })
  );
}

export async function createPolicy(apiKey: string, input: { id: string; name: string; sourceId: string; targetId: string }) {
  return policySchema.parse(
    await request("/api/v1/policies", apiKey, {
      method: "POST",
      body: JSON.stringify({
        ...input,
        protocol: "tcp",
        ports: ["*"],
        enabled: true
      })
    })
  );
}

export async function deletePolicy(apiKey: string, id: string) {
  await request(`/api/v1/policies/${encodeURIComponent(id)}`, apiKey, { method: "DELETE" });
}

export async function createInvitation(apiKey: string, input: { userId: string; groupId: string; reusable?: boolean }) {
  return invitationSchema.parse(
    await request("/api/v1/invitations", apiKey, {
      method: "POST",
      body: JSON.stringify({
        userId: input.userId,
        groupId: input.groupId,
        reusable: input.reusable ?? false,
        expiresInSeconds: 86400
      })
    })
  );
}

async function request(path: string, apiKey: string, init: RequestInit = {}) {
  const resp = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      "X-Linkbit-API-Key": apiKey,
      ...init.headers
    }
  });
  if (resp.status === 204) {
    return {};
  }
  const payload = await resp.json().catch(() => ({}));
  if (!resp.ok) {
    const message = typeof payload.error === "string" ? payload.error : resp.statusText;
    throw new Error(message);
  }
  return payload;
}
