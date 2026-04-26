import { Activity, Network, Server, ShieldCheck, SlidersHorizontal } from "lucide-react";
import { Button } from "./components/ui/button";
import { DashboardPage } from "./pages/DashboardPage";

const logoUrl = "/logo.svg";

const navItems = [
  { label: "仪表盘", icon: Activity },
  { label: "设备", icon: Network },
  { label: "中继", icon: Server },
  { label: "策略", icon: ShieldCheck },
  { label: "设置", icon: SlidersHorizontal }
];

export function App() {
  return (
    <main className="min-h-screen bg-background text-foreground">
      <aside className="fixed inset-y-0 left-0 hidden w-56 border-r border-border bg-white px-3 py-4 md:block">
        <div className="flex items-center gap-2 px-2 text-lg font-semibold">
          <img src={logoUrl} alt="Linkbit" className="h-8 w-8 rounded-md" />
          <span>Linkbit</span>
        </div>
        <nav className="mt-6 grid gap-1">
          {navItems.map((item) => (
            <Button key={item.label} variant="ghost" className="justify-start gap-2">
              <item.icon className="h-4 w-4" />
              {item.label}
            </Button>
          ))}
        </nav>
      </aside>
      <section className="md:pl-56">
        <DashboardPage />
      </section>
    </main>
  );
}
