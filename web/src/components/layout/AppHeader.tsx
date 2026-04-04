import { Separator } from "#/components/ui/separator";
import { SidebarTrigger } from "#/components/ui/sidebar";
import ThemeToggle from "../ThemeToggle";

export function AppHeader() {
	return (
		<header className="sticky top-0 z-50 flex h-14 shrink-0 items-center gap-2 border-b border-sidebar-border bg-sidebar px-4 backdrop-blur-sm">
			<SidebarTrigger className="-ml-1" />
			<Separator orientation="vertical" className="mr-2 h-4" />

			<div className="flex flex-1 items-center justify-between">
				<h1 className="text-lg font-semibold text-sidebar-foreground">
					Litchi
				</h1>

				<div className="flex items-center gap-2">
					<ThemeToggle />
				</div>
			</div>
		</header>
	);
}
