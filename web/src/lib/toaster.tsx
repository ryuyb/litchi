/**
 * Toast notification wrapper with theme integration.
 * Uses sonner for toast notifications with automatic theme detection.
 */
import { useEffect, useState } from "react";
import type { ToasterProps } from "sonner";
import { Toaster } from "#/components/ui/sonner";

type ThemeMode = "light" | "dark" | "system";

/**
 * Get resolved theme from document.documentElement class.
 * This is more efficient than polling localStorage.
 */
function getThemeFromDocument(): ThemeMode {
	const stored = localStorage.getItem("theme");
	if (stored === "light" || stored === "dark" || stored === "auto") {
		return stored === "auto" ? "system" : stored;
	}
	return "system";
}

/**
 * AppToaster - Toast notification component with theme support.
 * Wraps shadcn's Toaster with automatic theme detection.
 * Uses MutationObserver to detect theme changes efficiently.
 */
export function AppToaster(props?: ToasterProps) {
	const [theme, setTheme] = useState<ThemeMode>("system");

	useEffect(() => {
		// Initial theme
		setTheme(getThemeFromDocument());

		// Use MutationObserver to detect theme class changes on document
		// This is triggered by ThemeToggle without polling
		const observer = new MutationObserver(() => {
			setTheme(getThemeFromDocument());
		});

		observer.observe(document.documentElement, {
			attributes: true,
			attributeFilter: ["class", "data-theme"],
		});

		return () => observer.disconnect();
	}, []);

	return (
		<Toaster
			theme={theme}
			position="top-right"
			toastOptions={{
				classNames: {
					error:
						"!bg-destructive !text-destructive-foreground !border-destructive/50",
					success:
						"!bg-emerald-500/10 !text-emerald-600 dark:!text-emerald-400 !border-emerald-500/30",
					warning:
						"!bg-amber-500/10 !text-amber-600 dark:!text-amber-400 !border-amber-500/30",
					info: "!bg-primary/10 !text-primary !border-primary/30",
				},
			}}
			{...props}
		/>
	);
}

export { toast } from "sonner";
