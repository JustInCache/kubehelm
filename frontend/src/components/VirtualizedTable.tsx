import { FixedSizeList as List, type ListChildComponentProps } from 'react-window';
import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from '@tanstack/react-table';

type Props<T extends object> = {
  columns: ColumnDef<T>[];
  data: T[];
  height?: number;
};

export function VirtualizedTable<T extends object>({ columns, data, height = 460 }: Props<T>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  const rows = table.getRowModel().rows;

  return (
    <div className="card">
      <div style={{ display: 'grid', gridTemplateColumns: `repeat(${columns.length}, minmax(120px, 1fr))`, gap: 8, paddingBottom: 10 }}>
        {table.getFlatHeaders().map((header) => (
          <strong key={header.id}>{flexRender(header.column.columnDef.header, header.getContext())}</strong>
        ))}
      </div>
      <List height={height} itemCount={rows.length} itemSize={44} width="100%">
        {({ index, style }: ListChildComponentProps) => {
          const row = rows[index];
          return (
            <div style={{ ...style, display: 'grid', gridTemplateColumns: `repeat(${columns.length}, minmax(120px, 1fr))`, gap: 8, alignItems: 'center', borderTop: '1px solid #20304f', padding: '0 4px' }}>
              {row.getVisibleCells().map((cell) => (
                <div key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</div>
              ))}
            </div>
          );
        }}
      </List>
    </div>
  );
}

