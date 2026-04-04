import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type SortingState,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { DataTablePagination } from "./pagination";

interface DataTableProps<TData, TValue> {
	columns: ColumnDef<TData, TValue>[];
	data: TData[];
	pageCount?: number;
	pageIndex?: number;
	pageSize?: number;
	onPaginationChange?: (pageIndex: number, pageSize: number) => void;
	loading?: boolean;
}

export function DataTable<TData, TValue>({
	columns,
	data,
	pageCount,
	pageIndex = 0,
	pageSize = 10,
	onPaginationChange,
	loading = false,
}: DataTableProps<TData, TValue>) {
	const [sorting, setSorting] = useState<SortingState>([]);

	const table = useReactTable({
		data,
		columns,
		getCoreRowModel: getCoreRowModel(),
		onSortingChange: setSorting,
		state: {
			sorting,
		},
		// Server-side pagination
		manualPagination: !!onPaginationChange,
		pageCount: pageCount ?? Math.ceil(data.length / pageSize),
	});

	return (
		<div className="space-y-4">
			<div className="rounded-lg border border-border">
				<Table>
					<TableHeader>
						{table.getHeaderGroups().map((headerGroup) => (
							<TableRow key={headerGroup.id}>
								{headerGroup.headers.map((header) => (
									<TableHead key={header.id}>
										{header.isPlaceholder
											? null
											: flexRender(
													header.column.columnDef.header,
													header.getContext(),
												)}
									</TableHead>
								))}
							</TableRow>
						))}
					</TableHeader>
					<TableBody>
						{loading ? (
							<TableRow>
								<TableCell
									colSpan={columns.length}
									className="h-24 text-center"
								>
									<span className="text-muted-foreground">Loading...</span>
								</TableCell>
							</TableRow>
						) : table.getRowModel().rows?.length ? (
							table.getRowModel().rows.map((row) => (
								<TableRow
									key={row.id}
									data-state={row.getIsSelected() && "selected"}
								>
									{row.getVisibleCells().map((cell) => (
										<TableCell key={cell.id}>
											{flexRender(
												cell.column.columnDef.cell,
												cell.getContext(),
											)}
										</TableCell>
									))}
								</TableRow>
							))
						) : (
							<TableRow>
								<TableCell
									colSpan={columns.length}
									className="h-24 text-center"
								>
									<span className="text-muted-foreground">No results.</span>
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</Table>
			</div>

			{onPaginationChange && (
				<DataTablePagination
					table={table}
					pageIndex={pageIndex}
					pageSize={pageSize}
					onPaginationChange={onPaginationChange}
				/>
			)}
		</div>
	);
}
