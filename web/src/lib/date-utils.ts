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
