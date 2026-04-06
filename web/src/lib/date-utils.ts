// Format relative time from a date string
export function formatRelativeTime(dateStr: string | undefined): string {
	if (!dateStr) return "Unknown";

	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMins / 60);
	const diffDays = Math.floor(diffHours / 24);

	if (diffMins < 1) return "Just now";
	if (diffMins < 60) return `${diffMins} minutes ago`;
	if (diffHours < 24) return `${diffHours} hours ago`;
	if (diffDays < 7) return `${diffDays} days ago`;
	return date.toLocaleDateString();
}

// Format absolute datetime for display
export function formatDateTime(dateStr: string | undefined): string {
	if (!dateStr) return "-";

	const date = new Date(dateStr);
	return date.toLocaleString("en-US", {
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
		hour12: false,
	});
}

// Format date for RFC3339 query parameter
export function formatDateRFC3339(date: Date): string {
	return date.toISOString();
}
