import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Cpu, Gauge, KeyRound, RadioTower, Server } from "lucide-react";
import { useMemo, useState } from "react";
import { Button } from "../components/ui/button";
import {
  createInvitation,
  getDevices,
  getOverview,
  getPolicies,
  getRelays,
  getStoredAPIKey,
  storeAPIKey
} from "../lib/api";

const queryKeys = ["overview", "devices", "relays", "policies"] as const;

export function DashboardPage() {
  const queryClient = useQueryClient();
  const [apiKeyInput, setApiKeyInput] = useState(getStoredAPIKey);
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [lastToken, setLastToken] = useState("");
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
  const relays = useQuery({
    queryKey: ["relays", apiKey],
    queryFn: () => getRelays(apiKey),
    enabled
  });
  const policies = useQuery({
    queryKey: ["policies", apiKey],
    queryFn: () => getPolicies(apiKey),
    enabled
  });

  const invite = useMutation({
    mutationFn: () => createInvitation(apiKey),
    onSuccess: (value) => {
      setLastToken(value.token ?? "");
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
          <h2 className="text-base font-semibold">DERP 中继</h2>
          <div className="mt-4 grid gap-2">
            {(relays.data ?? []).length === 0 ? (
              <div className="rounded-md border border-dashed border-border p-6 text-sm text-muted-foreground">暂无中继节点</div>
            ) : (
              (relays.data ?? []).map((relay) => (
                <div key={relay.id} className="rounded-md border border-border p-3 text-sm">
                  <div className="font-medium">{relay.name}</div>
                  <div className="mt-1 text-muted-foreground">{relay.region} · {relay.status} · load {relay.load.toFixed(2)}</div>
                </div>
              ))
            )}
          </div>
        </div>
        <div className="rounded-lg border border-border bg-white p-4">
          <h2 className="text-base font-semibold">网络策略</h2>
          <div className="mt-4 grid gap-2">
            {(policies.data ?? []).length === 0 ? (
              <div className="rounded-md border border-dashed border-border p-6 text-sm text-muted-foreground">暂无策略</div>
            ) : (
              (policies.data ?? []).map((policy) => (
                <div key={policy.id} className="rounded-md border border-border p-3 text-sm">
                  <div className="font-medium">{policy.name}</div>
                  <div className="mt-1 text-muted-foreground">{policy.sourceId} → {policy.targetId}</div>
                </div>
              ))
            )}
          </div>
        </div>
      </section>
    </div>
  );
}
