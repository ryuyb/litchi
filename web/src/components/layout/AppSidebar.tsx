import { Link, useLocation } from "@tanstack/react-router";
import {
	CircleDotIcon,
	FileTextIcon,
	GitBranchIcon,
	LayoutDashboardIcon,
	SettingsIcon,
	TerminalIcon,
} from "lucide-react";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarRail,
	useSidebar,
} from "#/components/ui/sidebar";

const navItems = [
	{
		title: "Dashboard",
		url: "/",
		icon: LayoutDashboardIcon,
	},
	{
		title: "Issues",
		url: "/issues",
		icon: CircleDotIcon,
	},
	{
		title: "Repositories",
		url: "/repositories",
		icon: GitBranchIcon,
	},
	{
		title: "Audit Logs",
		url: "/audit-logs",
		icon: FileTextIcon,
	},
];

const settingsItem = {
	title: "Settings",
	url: "/settings",
	icon: SettingsIcon,
};

export function AppSidebar() {
	const location = useLocation();
	const { state } = useSidebar();
	const isCollapsed = state === "collapsed";

	return (
		<Sidebar
			collapsible="icon"
			className="border-r-border/30 shadow-[4px_0_24px_rgba(0,0,0,0.02)] bg-background/80 backdrop-blur-3xl"
		>
			<SidebarHeader className={isCollapsed ? "pt-6 pb-4" : "py-6 px-4"}>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton
							size="lg"
							asChild
							className={`hover:bg-transparent ${isCollapsed ? "justify-center" : ""}`}
						>
							<Link to="/" className="flex items-center gap-3">
								<div className="flex shrink-0 aspect-square size-10 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/80 text-primary-foreground shadow-lg shadow-primary/20 transition-transform group-hover:scale-105">
									<TerminalIcon className="size-5" />
								</div>
								{!isCollapsed && (
									<div className="flex flex-col gap-0.5 leading-none overflow-hidden animate-blur-in">
										<span className="font-bold text-lg tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-foreground to-foreground/70">
											Litchi
										</span>
										<span className="text-xs font-medium text-muted-foreground truncate">
											Dev Agent System
										</span>
									</div>
								)}
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarHeader>

			<SidebarContent className="gap-2">
				<SidebarGroup>
					{!isCollapsed && (
						<SidebarGroupLabel className="text-xs font-bold uppercase tracking-widest text-muted-foreground/70 mb-2">
							Workflow
						</SidebarGroupLabel>
					)}
					<SidebarGroupContent>
						<SidebarMenu className="gap-2">
							{navItems.map((item) => {
								const isActive =
									location.pathname === item.url ||
									(item.url !== "/" && location.pathname.startsWith(item.url));

								return (
									<SidebarMenuItem key={item.title}>
										<SidebarMenuButton
											asChild
											tooltip={item.title}
											className={`h-11 transition-all rounded-xl ${isCollapsed ? "justify-center" : "px-3"} ${isActive ? "bg-primary/10 text-primary hover:bg-primary/15" : "text-foreground/80 hover:bg-secondary/60 hover:text-foreground"}`}
										>
											<Link
												to={item.url}
												className="flex items-center gap-3 w-full"
											>
												<item.icon
													className={`shrink-0 size-5 ${isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground"}`}
												/>
												{!isCollapsed && (
													<span className="font-semibold">{item.title}</span>
												)}
												{!isCollapsed && isActive && (
													<div className="ml-auto shrink-0 w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
												)}
											</Link>
										</SidebarMenuButton>
									</SidebarMenuItem>
								);
							})}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>

			<SidebarFooter
				className={`border-t border-border/30 ${isCollapsed ? "py-4 flex justify-center items-center" : "p-4"}`}
			>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton
							asChild
							tooltip={settingsItem.title}
							className={`h-11 rounded-xl text-foreground/80 hover:bg-secondary/60 hover:text-foreground transition-all ${isCollapsed ? "justify-center" : "px-3"}`}
						>
							<Link
								to={settingsItem.url}
								className="flex items-center gap-3 w-full"
							>
								<settingsItem.icon className="shrink-0 size-5 text-muted-foreground group-hover:text-foreground" />
								{!isCollapsed && (
									<span className="font-semibold">{settingsItem.title}</span>
								)}
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarFooter>

			<SidebarRail />
		</Sidebar>
	);
}
