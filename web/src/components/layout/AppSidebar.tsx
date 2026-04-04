import { Link } from "@tanstack/react-router";
import {
	CircleDotIcon,
	FileTextIcon,
	GitBranchIcon,
	LayoutDashboardIcon,
	SettingsIcon,
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
} from "#/components/ui/sidebar";

// Navigation items for the sidebar
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
	return (
		<Sidebar collapsible="icon">
			<SidebarHeader>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton size="lg" asChild>
							<Link to="/">
								<div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
									<CircleDotIcon className="size-4" />
								</div>
								<div className="flex flex-col gap-0.5 leading-none">
									<span className="font-semibold">Litchi</span>
									<span className="text-xs text-muted-foreground">
										Dev Agent System
									</span>
								</div>
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarHeader>

			<SidebarContent>
				<SidebarGroup>
					<SidebarGroupLabel>Navigation</SidebarGroupLabel>
					<SidebarGroupContent>
						<SidebarMenu>
							{navItems.map((item) => (
								<SidebarMenuItem key={item.title}>
									<SidebarMenuButton asChild tooltip={item.title}>
										<Link to={item.url}>
											<item.icon />
											<span>{item.title}</span>
										</Link>
									</SidebarMenuButton>
								</SidebarMenuItem>
							))}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>

			<SidebarFooter>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton asChild tooltip={settingsItem.title}>
							<Link to={settingsItem.url}>
								<settingsItem.icon />
								<span>{settingsItem.title}</span>
							</Link>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarFooter>

			<SidebarRail />
		</Sidebar>
	);
}
