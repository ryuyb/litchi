import { createFileRoute } from "@tanstack/react-router";
import { LoaderIcon, ServerIcon, SettingsIcon } from "lucide-react";
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

// Type guard for successful response
function isSuccessResponse(
	response: { data: unknown; status: number } | undefined,
): response is { data: NonNullable<typeof response>["data"]; status: 200 } {
	return response?.status === 200;
}

function SettingsPage() {
	// Fetch config for read-only section
	const { data: configResponse, isLoading } = useGetApiV1Config();

	const config = isSuccessResponse(configResponse)
		? configResponse.data
		: undefined;

	return (
		<div className="space-y-6">
			{/* Header */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Settings
				</h1>
				<p className="mt-2 text-muted-foreground">
					Configure system preferences and integrations.
				</p>
			</section>

			{/* Settings Tabs */}
			<Tabs defaultValue="general" className="space-y-6">
				<TabsList variant="line" className="bg-transparent">
					<TabsTrigger value="general">
						<SettingsIcon className="size-4" />
						<span>General</span>
					</TabsTrigger>
					<TabsTrigger value="system">
						<ServerIcon className="size-4" />
						<span>System</span>
					</TabsTrigger>
				</TabsList>

				<TabsContent value="general" className="space-y-6">
					<SettingsForm />
				</TabsContent>

				<TabsContent value="system" className="space-y-6">
					{isLoading ? (
						<div className="flex items-center justify-center min-h-[200px] gap-2">
							<LoaderIcon className="size-6 animate-spin text-muted-foreground" />
							<span className="text-muted-foreground">
								Loading system info...
							</span>
						</div>
					) : (
						<>
							<ReadOnlyConfigSection config={config} />
							<HealthStatusSection />
						</>
					)}
				</TabsContent>
			</Tabs>
		</div>
	);
}
