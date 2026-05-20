# Table Rendering Sample

Exercises the gopdf backend's table renderer: multi-line cell wrapping, mixed
column widths, and page breaks that fall inside a table.

## 1. Short single-line table

A baseline check — every cell fits on one line, so every row should be the
same height.

| Code | Name      | Latency |
|------|-----------|---------|
| NOP  | No-op     | 1       |
| ADD  | Add       | 1       |
| MUL  | Multiply  | 3       |
| DIV  | Divide    | 12      |
| LD   | Load      | 4       |
| ST   | Store     | 4       |

## 2. Mixed widths with multi-line descriptions

The Description column has prose of varying lengths so different rows wrap
to 1, 2, or 3 lines. Row heights should adapt so the tallest cell in each
row sets the row height, and the borders stay aligned across columns.

Each row is annotated with its expected line count for easy verification.

| Code | Name        | Description |
|------|-------------|-------------|
| NOP  | No-op       | _(1 line)_ Advances PC without modifying state. |
| ADD  | Add         | _(2 lines)_ Adds two source registers, writes the result to the destination, and sets the carry flag on overflow. |
| MUL  | Multiply    | _(3 lines)_ Computes the product of two source registers as a value up to twice the operand width. The high half is discarded unless the wide-result variant is used. |
| DIV  | Divide      | _(1 line)_ Integer divide with truncation toward zero. |
| LD   | Load        | _(2 lines)_ Reads a word from memory at the effective address formed by the base register plus an immediate offset. |
| ST   | Store       | _(3 lines)_ Writes a word to memory at the effective address. Honors the same alignment constraints as the corresponding load, faulting on misaligned addresses. |

## 3. Wider table that exercises the width algorithm

Three columns of comparable but unequal natural widths — the CSS-style
min/max distribution should leave none of them obviously cramped.

| Category         | Examples                         | Typical use                                                       |
|------------------|----------------------------------|-------------------------------------------------------------------|
| Arithmetic       | `ADD`, `SUB`, `MUL`, `DIV`       | General numeric computation in loops and expressions.             |
| Logical          | `AND`, `OR`, `XOR`, `NOT`        | Bit manipulation, mask construction, predicate evaluation.        |
| Memory           | `LD`, `ST`, `LDX`, `STX`         | Moving data between registers and memory; spilling and reloading. |
| Control flow     | `BEQ`, `BNE`, `JMP`, `CALL`      | Conditional and unconditional branches; function call sequences.  |
| Synchronization  | `FENCE`, `LL`, `SC`              | Ordering memory operations and implementing atomics.              |

## 4. Long table that crosses a page boundary

The preamble below is sized to push the table about two-thirds of the way
down a page so the table _must_ cross into the next page mid-flight. The
renderer should break between rows (never inside a cell), and the body
content immediately after should continue normally on the page where the
table finishes.

### Preamble (to position the table)

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim
veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
commodo consequat. Duis aute irure dolor in reprehenderit in voluptate
velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat
cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id
est laborum.

Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium
doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo
inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.
Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut
fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem
sequi nesciunt.

Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet,
consectetur, adipisci velit, sed quia non numquam eius modi tempora
incidunt ut labore et dolore magnam aliquam quaerat voluptatem.

### Reference table

| ID  | Operation     | Description                                                                                                                       |
|-----|---------------|-----------------------------------------------------------------------------------------------------------------------------------|
| 001 | Bootstrap     | Initial hardware reset sequence; clears the register file, primes the instruction cache, and jumps to the reset vector.            |
| 002 | Configure     | Loads architectural configuration words into the control registers, enabling features required by the boot loader.                |
| 003 | Self-test     | Runs the on-die self-test pattern across the SRAM banks and reports any single-bit or double-bit errors detected.                 |
| 004 | Calibrate     | Walks the DDR PHY through write-leveling and read-gate training; records the resulting timings in non-volatile storage.           |
| 005 | Enumerate     | Scans the attached peripheral bus, identifies devices by vendor and product ID, and populates the device tree.                    |
| 006 | Authenticate  | Verifies the next-stage loader against the on-die public key; refuses to continue on signature mismatch.                          |
| 007 | Decompress    | Streams the compressed kernel image out of NOR flash, decompresses it in place, and verifies the post-decompression checksum.     |
| 008 | Relocate      | Copies the now-decompressed kernel to its final load address and patches any position-dependent jump targets.                     |
| 009 | Hand off      | Sets up the initial stack pointer, page table base, and exception vector base, then branches into the kernel entry point.         |
| 010 | Idle          | Park the boot processor in a low-power wait-for-interrupt state until application processors signal readiness.                    |
| 011 | Wake          | Coordinated boot of secondary cores; each acknowledges via the inter-processor mailbox before joining the active set.             |
| 012 | Schedule      | Hand control to the task scheduler with the initial workload queued; further behavior is OS-dependent.                            |
| 013 | Quiesce       | Pause non-essential subsystems ahead of a checkpoint; flush write buffers and confirm caches are in a consistent state.            |
| 014 | Checkpoint    | Snapshot architectural state and durable-storage offsets into the recovery region for fast restart.                                |
| 015 | Resume        | Restore architectural state from the most recent checkpoint and re-enter the scheduler with the saved workload queue.             |
| 016 | Shutdown      | Coordinated power-down of all cores and peripherals; persists any pending writes and releases external resets.                    |

### After the table

A paragraph below the table to confirm the renderer leaves the cursor in a
sensible position once table rendering finishes — no overlap with the last
row's border, no spurious blank page, and inline formatting still works:
**bold**, *italic*, `inline code`, and a [link](https://example.com/).
