import { useLocation } from "@tanstack/react-router";
import { Separator } from "#/components/ui/separator";
import { SidebarTrigger } from "#/components/ui/sidebar";
import ThemeToggle from "../ThemeToggle";

// Navigation formatting map
const pathLabels: Record<string, string> = {
	"/": "Dashboard",
	"/issues": "Issues",
	"/repositories": "Repositories",
	"/audit-logs": "Audit Logs",
	"/settings": "Settings",
};

export function AppHeader() {
	const location = useLocation();

	// Try to get a matching label, or fallback to parsing the path
	const getPathLabel = () => {
		if (pathLabels[location.pathname]) {
			return pathLabels[location.pathname];
		}

		const parts = location.pathname.split("/").filter(Boolean);
		if (parts.length > 0) {
			const label = parts[0];
			return label.charAt(0).toUpperCase() + label.slice(1).replace("-", " ");
		}

		return "Dashboard";
	};

	return (
		<header className="sticky top-0 z-50 flex h-16 shrink-0 items-center justify-between border-b border-border/40 bg-background/60 backdrop-blur-xl px-6 supports-[backdrop-filter]:bg-background/40">
			<div className="flex items-center gap-4">
				<SidebarTrigger className="hover:bg-secondary/80 bg-secondary/30 rounded-xl size-9 transition-colors" />
				<Separator
					orientation="vertical"
					className="h-6 w-[1px] bg-border/60"
				/>
				<div className="flex items-center gap-2 text-sm font-medium text-muted-foreground animate-slide-up-fade">
					<span className="text-foreground font-semibold px-2 py-1 rounded-lg bg-secondary/50 border border-border/30 shadow-sm">
						{getPathLabel()}
					</span>
				</div>
			</div>

			<div className="flex items-center gap-3">
				<div className="h-9 flex items-center justify-center rounded-xl bg-secondary/30 border border-border/50 px-3 hover:bg-secondary/50 transition-colors cursor-pointer mr-2 shadow-sm">
					<span className="relative flex h-2 w-2 mr-2">
						<span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
						<span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
					</span>
					<span className="text-xs font-bold text-foreground">
						Litchi Agent Active
					</span>
				</div>
				<Separator
					orientation="vertical"
					className="h-6 w-[1px] bg-border/60 mr-1"
				/>
				<ThemeToggle />
			</div>
		</header>
	);
}
