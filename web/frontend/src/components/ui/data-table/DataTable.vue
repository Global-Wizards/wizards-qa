<script setup>
import { FlexRender, useVueTable, getCoreRowModel, getSortedRowModel, getFilteredRowModel } from '@tanstack/vue-table'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const props = defineProps({
  columns: { type: Array, required: true },
  data: { type: Array, required: true },
  sorting: { type: Array, default: () => [] },
  globalFilter: { type: String, default: '' },
  rowSelection: { type: Object, default: () => ({}) },
  onRowClick: { type: Function, default: null },
  emptyText: { type: String, default: 'No results.' },
})

const emit = defineEmits(['update:sorting', 'update:rowSelection', 'update:globalFilter'])

const table = useVueTable({
  get data() { return props.data },
  get columns() { return props.columns },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getFilteredRowModel: getFilteredRowModel(),
  state: {
    get sorting() { return props.sorting },
    get globalFilter() { return props.globalFilter },
    get rowSelection() { return props.rowSelection },
  },
  onSortingChange: (updater) => {
    const val = typeof updater === 'function' ? updater(props.sorting) : updater
    emit('update:sorting', val)
  },
  onRowSelectionChange: (updater) => {
    const val = typeof updater === 'function' ? updater(props.rowSelection) : updater
    emit('update:rowSelection', val)
  },
  onGlobalFilterChange: (updater) => {
    const val = typeof updater === 'function' ? updater(props.globalFilter) : updater
    emit('update:globalFilter', val)
  },
  enableRowSelection: true,
})

defineExpose({ table })
</script>

<template>
  <Table>
    <TableHeader>
      <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id">
        <TableHead v-for="header in headerGroup.headers" :key="header.id" :class="header.column.columnDef.meta?.class">
          <FlexRender
            v-if="!header.isPlaceholder"
            :render="header.column.columnDef.header"
            :props="header.getContext()"
          />
        </TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      <template v-if="table.getRowModel().rows.length">
        <TableRow
          v-for="row in table.getRowModel().rows"
          :key="row.id"
          :data-state="row.getIsSelected() ? 'selected' : undefined"
          :class="onRowClick ? 'cursor-pointer' : ''"
          @click="onRowClick && onRowClick(row)"
        >
          <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id" :class="cell.column.columnDef.meta?.class">
            <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
          </TableCell>
        </TableRow>
      </template>
      <template v-else>
        <TableRow>
          <TableCell :colspan="columns.length" class="h-24 text-center text-muted-foreground">
            <slot name="empty">
              {{ emptyText }}
            </slot>
          </TableCell>
        </TableRow>
      </template>
    </TableBody>
  </Table>
</template>
