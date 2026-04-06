import { createFileRoute } from "@tanstack/react-router";
import {
	LoaderIcon,
	ServerIcon,
	SettingsIcon,
	ShieldCheckIcon,
} from "lucide-react";
import { useGetApiV1Config } from "#/api/config/config";
import {
	HealthStatusSection,
	ReadOnlyConfigSection,
	SettingsForm,
} from "#/components/settings";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";

export const Route = createFileRoute("/settings/")({
	component: SettingsPage,
});

function isSuccessResponse(
	response: { data: unknown; status: number } | undefined,
): response is { data: NonNullable<typeof response>["data"]; status: 200 } {
	return response?.status === 200;
}

function SettingsPage() {
	const { data: configResponse, isLoading } = useGetApiV1Config();

	const config = isSuccessResponse(configResponse)
		? configResponse.data
		: undefined;

	return (
		<div className="space-y-8 animate-blur-in w-full pb-10">
			{/* Header */}
			<section className="relative overflow-hidden rounded-3xl bg-gradient-to-l from-slate-800 to-slate-900 p-8 shadow-xl text-white">
				<div className="absolute left-0 top-0 w-full h-full bg-[url('https://transparenttextures.com/patterns/black-scales.png')] opacity-20"></div>
				<div className="absolute right-0 bottom-0 p-8 opacity-10">
					<SettingsIcon size={180} />
				</div>
				<div className="relative z-10">
					<h1 className="text-3xl font-bold tracking-tight flex items-center gap-3">
						<span className="w-1.5 h-8 bg-blue-500 rounded-full"></span>
						Settings
					</h1>
					<p className="mt-3 text-slate-300 max-w-2xl text-lg">
						Configure system preferences, user integrations, and monitor core
						application health.
					</p>
				</div>
			</section>

			{/* Settings Tabs */}
			<div className="glass-card rounded-3xl p-6 shadow-md border border-border/50">
				<Tabs defaultValue="general" className="space-y-8">
					<TabsList className="bg-secondary/50 p-1.5 rounded-2xl border border-border/30 inline-flex">
						<TabsTrigger
							value="general"
							className="rounded-xl px-6 py-2.5 data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:font-bold font-medium transition-all"
						>
							<div className="flex items-center gap-2">
								<SettingsIcon className="size-4" />
								<span>General</span>
							</div>
						</TabsTrigger>
						<TabsTrigger
							value="system"
							className="rounded-xl px-6 py-2.5 data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:font-bold font-medium transition-all"
						>
							<div className="flex items-center gap-2">
								<ServerIcon className="size-4" />
								<span>System & Health</span>
							</div>
						</TabsTrigger>
					</TabsList>

					<TabsContent
						value="general"
						className="space-y-6 animate-slide-up-fade text-foreground"
					>
						<div className="p-2">
							<SettingsForm />
						</div>
					</TabsContent>

					<TabsContent
						value="system"
						className="space-y-8 animate-slide-up-fade text-foreground"
					>
						{isLoading ? (
							<div className="flex flex-col items-center justify-center min-h-[300px] gap-4 bg-secondary/10 rounded-2xl border border-dashed border-border p-10">
								<div className="p-4 bg-background rounded-full shadow-sm">
									<LoaderIcon className="size-8 animate-spin text-primary" />
								</div>
								<span className="text-muted-foreground font-medium text-lg">
									Loading system diagnostics...
								</span>
							</div>
						) : (
							<div className="grid gap-8">
								<div className="glass-card bg-background/50 border border-border/60 rounded-2xl p-6 shadow-sm">
									<div className="flex items-center gap-3 mb-6 border-b border-border/50 pb-4">
										<div className="bg-blue-500/10 text-blue-500 p-2 rounded-lg">
											<ServerIcon size={20} />
										</div>
										<h2 className="text-xl font-bold">Configuration</h2>
									</div>
									<ReadOnlyConfigSection config={config} />
								</div>

								<div className="glass-card bg-background/50 border border-border/60 rounded-2xl p-6 shadow-sm">
									<div className="flex items-center gap-3 mb-6 border-b border-border/50 pb-4">
										<div className="bg-emerald-500/10 text-emerald-500 p-2 rounded-lg">
											<ShieldCheckIcon size={20} />
										</div>
										<h2 className="text-xl font-bold">Health Status</h2>
									</div>
									<HealthStatusSection />
								</div>
							</div>
						)}
					</TabsContent>
				</Tabs>
			</div>
		</div>
	);
}
