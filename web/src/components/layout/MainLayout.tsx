import type { ReactNode } from "react";
import { SidebarInset, SidebarProvider } from "#/components/ui/sidebar";
import { AppHeader } from "./AppHeader";
import { AppSidebar } from "./AppSidebar";

interface MainLayoutProps {
	children: ReactNode;
}

export function MainLayout({ children }: MainLayoutProps) {
	return (
		<SidebarProvider defaultOpen>
			<AppSidebar />
			<SidebarInset>
				<AppHeader />
				<main className="flex flex-1 flex-col gap-6 p-4 md:p-8 lg:px-12 pt-6 w-full max-w-full overflow-x-hidden">{children}</main>
			</SidebarInset>
		</SidebarProvider>
	);
}
