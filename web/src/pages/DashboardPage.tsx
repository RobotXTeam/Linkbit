import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Cpu, Gauge, KeyRound, Plus, RadioTower, Server, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { Button } from "../components/ui/button";
import {
  createAPIKey,
  createGroup,
  createInvitation,
  createPolicy,
  createUser,
  deletePolicy,
  deleteRelay,
  getDERPMap,
  getAPIKeys,
  getDevices,
  getGroups,
  getOverview,
  getPolicies,
  getRelays,
  getStoredAPIKey,
  getUsers,
  registerRelay,
  revokeAPIKey,
  storeAPIKey
} from "../lib/api";

const queryKeys = ["overview", "devices", "relays", "policies", "apiKeys", "users", "groups"] as const;

export function DashboardPage() {
  const queryClient = useQueryClient();
  const [apiKeyInput, setApiKeyInput] = useState(getStoredAPIKey);
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [lastToken, setLastToken] = useState("");
  const [lastAPIKey, setLastAPIKey] = useState("");
  const [userForm, setUserForm] = useState({ id: "default-user", name: "Default User", email: "", role: "member" });
  const [groupForm, setGroupForm] = useState({ id: "default", name: "Default", description: "" });
  const [policyForm, setPolicyForm] = useState({ id: "", name: "", sourceId: "*", targetId: "default" });
  const [inviteForm, setInviteForm] = useState({ userId: "default-user", groupId: "default", reusable: false });
  const [relayForm, setRelayForm] = useState({
    id: "",
    name: "",
    region: "default",
    publicUrl: ""
  });
  const enabled = apiKey.length > 0;

  const overview = useQuery({
    queryKey: ["overview", apiKey],
    queryFn: () => getOverview(apiKey),
    enabled
  });
  const devices = useQuery({
    queryKey: ["devices", apiKey],
    queryFn: () => getDevices(apiKey),
    enabled
  });
  const users = useQuery({
    queryKey: ["users", apiKey],
    queryFn: () => getUsers(apiKey),
    enabled
  });
  const groups = useQuery({
    queryKey: ["groups", apiKey],
    queryFn: () => getGroups(apiKey),
    enabled
  });
  const relays = useQuery({
    queryKey: ["relays", apiKey],
    queryFn: () => getRelays(apiKey),
    enabled
  });
  const derpMap = useQuery({
    queryKey: ["derpMap", apiKey],
    queryFn: () => getDERPMap(apiKey),
    enabled
  });
  const policies = useQuery({
    queryKey: ["policies", apiKey],
    queryFn: () => getPolicies(apiKey),
    enabled
  });
  const apiKeys = useQuery({
    queryKey: ["apiKeys", apiKey],
    queryFn: () => getAPIKeys(apiKey),
    enabled
  });

  const invite = useMutation({
    mutationFn: () => createInvitation(apiKey, inviteForm),
    onSuccess: (value) => {
      setLastToken(value.token ?? "");
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const addUser = useMutation({
    mutationFn: () => createUser(apiKey, userForm),
    onSuccess: () => {
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const addGroup = useMutation({
    mutationFn: () => createGroup(apiKey, groupForm),
    onSuccess: () => {
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const addPolicy = useMutation({
    mutationFn: () => createPolicy(apiKey, { ...policyForm, id: policyForm.id || crypto.randomUUID() }),
    onSuccess: () => {
      setPolicyForm({ id: "", name: "", sourceId: "*", targetId: "default" });
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const createRelayKey = useMutation({
    mutationFn: () => createAPIKey(apiKey, "relay"),
    onSuccess: (value) => {
      setLastAPIKey(value.key ?? "");
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const addRelay = useMutation({
    mutationFn: () => registerRelay(apiKey, relayForm),
    onSuccess: () => {
      setRelayForm({ id: "", name: "", region: "default", publicUrl: "" });
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const removeRelay = useMutation({
    mutationFn: (id: string) => deleteRelay(apiKey, id),
    onSuccess: () => {
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const removePolicy = useMutation({
    mutationFn: (id: string) => deletePolicy(apiKey, id),
    onSuccess: () => {
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });
  const revokeKey = useMutation({
    mutationFn: (id: string) => revokeAPIKey(apiKey, id),
    onSuccess: () => {
      queryKeys.forEach((key) => void queryClient.invalidateQueries({ queryKey: [key, apiKey] }));
    }
  });

  const stats = useMemo(
    () => [
      { label: "在线设备", value: String(overview.data?.onlineDevices ?? 0), icon: Cpu },
      { label: "中继节点", value: String(overview.data?.relayNodes ?? 0), icon: RadioTower },
      { label: "网络健康度", value: overview.data?.networkHealth ?? "未连接", icon: Gauge },
      { label: "策略数量", value: String(overview.data?.policyCount ?? 0), icon: Server }
    ],
    [overview.data]
  );

  const saveKey = () => {
    storeAPIKey(apiKeyInput);
    setApiKey(apiKeyInput.trim());
  };

  return (
    <div className="mx-auto max-w-7xl px-5 py-6">
      <header className="flex flex-col gap-3 border-b border-border pb-5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-normal">控制台</h1>
          <p className="mt-1 text-sm text-muted-foreground">设备、策略和中继节点的运行视图</p>
        </div>
        <div className="flex w-full flex-col gap-2 sm:w-auto sm:min-w-96 sm:flex-row">
          <div className="relative flex-1">
            <KeyRound className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <input
              className="h-9 w-full rounded-md border border-border bg-white pl-9 pr-3 text-sm outline-none focus:ring-2 focus:ring-primary"
              placeholder="Admin API Key"
              type="password"
              value={apiKeyInput}
              onChange={(event) => setApiKeyInput(event.target.value)}
            />
          </div>
          <Button onClick={saveKey}>连接</Button>
        </div>
      </header>

      <section className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => (
          <div key={stat.label} className="rounded-lg border border-border bg-white p-4">
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm text-muted-foreground">{stat.label}</span>
              <stat.icon className="h-4 w-4 text-muted-foreground" />
            </div>
            <div className="mt-3 text-2xl font-semibold">{stat.value}</div>
          </div>
        ))}
      </section>

      {overview.error instanceof Error ? (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {overview.error.message}
        </div>
      ) : null}

      <section className="mt-6 grid gap-4 lg:grid-cols-[1.4fr_1fr]">
        <div className="rounded-lg border border-border bg-white p-4">
          <h2 className="text-base font-semibold">最近设备</h2>
          <div className="mt-4 overflow-hidden rounded-md border border-border">
            {(devices.data ?? []).length === 0 ? (
              <div className="p-6 text-sm text-muted-foreground">暂无设备</div>
            ) : (
              (devices.data ?? []).map((device) => (
                <div key={device.id} className="grid gap-1 border-b border-border p-3 text-sm last:border-b-0">
                  <div className="font-medium">{device.name}</div>
                  <div className="text-muted-foreground">{device.virtualIp} · {device.status}</div>
                </div>
              ))
            )}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-white p-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">设备邀请</h2>
            <Button onClick={() => invite.mutate()} disabled={!enabled || invite.isPending}>
              生成
            </Button>
          </div>
          <div className="mt-4 grid gap-2 sm:grid-cols-2">
            <input
              className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
              placeholder="用户 ID"
              value={inviteForm.userId}
              onChange={(event) => setInviteForm((value) => ({ ...value, userId: event.target.value }))}
            />
            <input
              className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
              placeholder="设备组 ID"
              value={inviteForm.groupId}
              onChange={(event) => setInviteForm((value) => ({ ...value, groupId: event.target.value }))}
            />
          </div>
          {lastToken ? (
            <button
              className="mt-4 flex w-full items-center justify-between gap-3 rounded-md border border-border bg-muted p-3 text-left text-xs"
              onClick={() => void navigator.clipboard.writeText(lastToken)}
            >
              <span className="break-all">{lastToken}</span>
              <Copy className="h-4 w-4 shrink-0" />
            </button>
          ) : (
            <div className="mt-4 rounded-md border border-dashed border-border p-6 text-sm text-muted-foreground">
              生成后只显示一次。
            </div>
          )}
        </div>
      </section>

      <section className="mt-6 grid gap-4 lg:grid-cols-2">
        <div className="rounded-lg border border-border bg-white p-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">服务器 / DERP 中继</h2>
            <Button onClick={() => createRelayKey.mutate()} disabled={!enabled || createRelayKey.isPending}>
              中继密钥
            </Button>
          </div>
          <form
            className="mt-4 grid gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              addRelay.mutate();
            }}
          >
            <div className="grid gap-2 sm:grid-cols-2">
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
                placeholder="服务器 ID"
                value={relayForm.id}
                onChange={(event) => setRelayForm((value) => ({ ...value, id: event.target.value }))}
              />
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
                placeholder="名称"
                value={relayForm.name}
                onChange={(event) => setRelayForm((value) => ({ ...value, name: event.target.value }))}
              />
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
                placeholder="区域"
                value={relayForm.region}
                onChange={(event) => setRelayForm((value) => ({ ...value, region: event.target.value }))}
              />
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
                placeholder="公网 URL"
                value={relayForm.publicUrl}
                onChange={(event) => setRelayForm((value) => ({ ...value, publicUrl: event.target.value }))}
              />
            </div>
            <Button className="w-fit gap-2" disabled={!enabled || addRelay.isPending}>
              <Plus className="h-4 w-4" />
              添加
            </Button>
          </form>
          {lastAPIKey ? (
            <button
              className="mt-4 flex w-full items-center justify-between gap-3 rounded-md border border-border bg-muted p-3 text-left text-xs"
              onClick={() => void navigator.clipboard.writeText(lastAPIKey)}
            >
              <span className="break-all">{lastAPIKey}</span>
              <Copy className="h-4 w-4 shrink-0" />
            </button>
          ) : null}
          <div className="mt-4 rounded-md border border-border bg-muted p-3 text-sm text-muted-foreground">
            DERP map 区域数：{Object.keys(derpMap.data?.Regions ?? {}).length}
          </div>
          <div className="mt-4 grid gap-2">
            {(relays.data ?? []).length === 0 ? (
              <div className="rounded-md border border-dashed border-border p-6 text-sm text-muted-foreground">暂无中继节点</div>
            ) : (
              (relays.data ?? []).map((relay) => (
                <div key={relay.id} className="flex items-start justify-between gap-3 rounded-md border border-border p-3 text-sm">
                  <div>
                    <div className="font-medium">{relay.name}</div>
                    <div className="mt-1 text-muted-foreground">{relay.region} · {relay.status} · load {relay.load.toFixed(2)}</div>
                    <div className="mt-1 break-all text-xs text-muted-foreground">{relay.publicUrl}</div>
                  </div>
                  <Button variant="ghost" onClick={() => removeRelay.mutate(relay.id)}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))
            )}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-white p-4">
          <h2 className="text-base font-semibold">网络策略</h2>
          <form
            className="mt-4 grid gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              addPolicy.mutate();
            }}
          >
            <div className="grid gap-2 sm:grid-cols-2">
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
                placeholder="来源设备/组/*"
                value={policyForm.sourceId}
                onChange={(event) => setPolicyForm((value) => ({ ...value, sourceId: event.target.value }))}
              />
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary"
                placeholder="目标设备/组"
                value={policyForm.targetId}
                onChange={(event) => setPolicyForm((value) => ({ ...value, targetId: event.target.value }))}
              />
              <input
                className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary sm:col-span-2"
                placeholder="策略名称"
                value={policyForm.name}
                onChange={(event) => setPolicyForm((value) => ({ ...value, name: event.target.value }))}
              />
            </div>
            <Button className="w-fit gap-2" disabled={!enabled || addPolicy.isPending}>
              <Plus className="h-4 w-4" />
              添加策略
            </Button>
          </form>
          <div className="mt-4 grid gap-2">
            {(policies.data ?? []).length === 0 ? (
              <div className="rounded-md border border-dashed border-border p-6 text-sm text-muted-foreground">暂无策略</div>
            ) : (
              (policies.data ?? []).map((policy) => (
                <div key={policy.id} className="flex items-start justify-between gap-3 rounded-md border border-border p-3 text-sm">
                  <div>
                    <div className="font-medium">{policy.name}</div>
                    <div className="mt-1 text-muted-foreground">{policy.sourceId} → {policy.targetId}</div>
                  </div>
                  <Button variant="ghost" onClick={() => removePolicy.mutate(policy.id)}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))
            )}
          </div>
        </div>
      </section>

      <section className="mt-6 grid gap-4 lg:grid-cols-2">
        <div className="rounded-lg border border-border bg-white p-4">
          <h2 className="text-base font-semibold">用户</h2>
          <form
            className="mt-4 grid gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              addUser.mutate();
            }}
          >
            <div className="grid gap-2 sm:grid-cols-2">
              <input className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary" placeholder="用户 ID" value={userForm.id} onChange={(event) => setUserForm((value) => ({ ...value, id: event.target.value }))} />
              <input className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary" placeholder="名称" value={userForm.name} onChange={(event) => setUserForm((value) => ({ ...value, name: event.target.value }))} />
              <input className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary" placeholder="邮箱" value={userForm.email} onChange={(event) => setUserForm((value) => ({ ...value, email: event.target.value }))} />
              <select className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary" value={userForm.role} onChange={(event) => setUserForm((value) => ({ ...value, role: event.target.value }))}>
                <option value="member">member</option>
                <option value="admin">admin</option>
              </select>
            </div>
            <Button className="w-fit gap-2" disabled={!enabled || addUser.isPending}>
              <Plus className="h-4 w-4" />
              添加用户
            </Button>
          </form>
          <div className="mt-4 grid gap-2">
            {(users.data ?? []).map((user) => (
              <div key={user.id} className="rounded-md border border-border p-3 text-sm">
                <div className="font-medium">{user.name}</div>
                <div className="text-muted-foreground">{user.id} · {user.role}</div>
              </div>
            ))}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-white p-4">
          <h2 className="text-base font-semibold">设备组</h2>
          <form
            className="mt-4 grid gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              addGroup.mutate();
            }}
          >
            <div className="grid gap-2 sm:grid-cols-2">
              <input className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary" placeholder="组 ID" value={groupForm.id} onChange={(event) => setGroupForm((value) => ({ ...value, id: event.target.value }))} />
              <input className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary" placeholder="名称" value={groupForm.name} onChange={(event) => setGroupForm((value) => ({ ...value, name: event.target.value }))} />
              <input className="h-9 rounded-md border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary sm:col-span-2" placeholder="描述" value={groupForm.description} onChange={(event) => setGroupForm((value) => ({ ...value, description: event.target.value }))} />
            </div>
            <Button className="w-fit gap-2" disabled={!enabled || addGroup.isPending}>
              <Plus className="h-4 w-4" />
              添加设备组
            </Button>
          </form>
          <div className="mt-4 grid gap-2">
            {(groups.data ?? []).map((group) => (
              <div key={group.id} className="rounded-md border border-border p-3 text-sm">
                <div className="font-medium">{group.name}</div>
                <div className="text-muted-foreground">{group.id}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mt-6 rounded-lg border border-border bg-white p-4">
        <h2 className="text-base font-semibold">API Key</h2>
        <div className="mt-4 grid gap-2">
          {(apiKeys.data ?? []).length === 0 ? (
            <div className="rounded-md border border-dashed border-border p-6 text-sm text-muted-foreground">暂无持久密钥</div>
          ) : (
            (apiKeys.data ?? []).map((item) => (
              <div key={item.id} className="flex items-start justify-between gap-3 rounded-md border border-border p-3 text-sm">
                <div>
                  <div className="font-medium">{item.name}</div>
                  <div className="text-muted-foreground">
                    {item.scope} · {new Date(item.createdAt).toLocaleString()}
                    {item.revokedAt ? " · revoked" : ""}
                  </div>
                </div>
                {!item.revokedAt ? (
                  <Button variant="ghost" onClick={() => revokeKey.mutate(item.id)}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                ) : null}
              </div>
            ))
          )}
        </div>
      </section>
    </div>
  );
}
