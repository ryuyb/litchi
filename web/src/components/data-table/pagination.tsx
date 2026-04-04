import type { Table } from "@tanstack/react-table";
import {
	Pagination,
	PaginationContent,
	PaginationItem,
	PaginationNext,
	PaginationPrevious,
} from "#/components/ui/pagination";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";

interface DataTablePaginationProps<TData> {
	table: Table<TData>;
	pageIndex: number;
	pageSize: number;
	onPaginationChange: (pageIndex: number, pageSize: number) => void;
}

export function DataTablePagination<TData>({
	pageIndex,
	pageSize,
	onPaginationChange,
}: DataTablePaginationProps<TData>) {
	const canPreviousPage = pageIndex > 0;

	return (
		<div className="flex items-center justify-between">
			<div className="flex items-center space-x-2">
				<p className="text-sm text-muted-foreground">Rows per page</p>
				<Select
					value={`${pageSize}`}
					onValueChange={(value) => {
						onPaginationChange(0, Number(value));
					}}
				>
					<SelectTrigger className="h-8 w-[70px]">
						<SelectValue placeholder={pageSize} />
					</SelectTrigger>
					<SelectContent side="top">
						{[10, 20, 30, 40, 50].map((size) => (
							<SelectItem key={size} value={`${size}`}>
								{size}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
			</div>
			<Pagination>
				<PaginationContent>
					<PaginationItem>
						<PaginationPrevious
							onClick={() => onPaginationChange(pageIndex - 1, pageSize)}
							className={
								!canPreviousPage
									? "pointer-events-none opacity-50"
									: "cursor-pointer"
							}
						/>
					</PaginationItem>
					<PaginationItem>
						<span className="flex h-9 w-9 items-center justify-center text-sm">
							Page {pageIndex + 1}
						</span>
					</PaginationItem>
					<PaginationItem>
						<PaginationNext
							onClick={() => onPaginationChange(pageIndex + 1, pageSize)}
							className="cursor-pointer"
						/>
					</PaginationItem>
				</PaginationContent>
			</Pagination>
		</div>
	);
}
