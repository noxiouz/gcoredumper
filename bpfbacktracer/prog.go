package bpfbacktracer

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"path/filepath"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	"golang.org/x/sys/unix"
)

const (
	GCoreSamlesMapName = "gcored_samples"
	FramesNumber       = 4
)

var gCoreSamlesMapSpec = &ebpf.MapSpec{
	Name:       GCoreSamlesMapName,
	Type:       ebpf.Hash,
	KeySize:    4,                // u32 PID
	ValueSize:  FramesNumber * 8, // 4 frames
	MaxEntries: 128,
}

type BpfBacktracer struct {
	samplesMap *ebpf.Map
	prog       *ebpf.Program
	link       link.Link
}

/*
	TODO: use bpf2go to generate a program
	BCC version of a program:
prog.c
```
#include <uapi/linux/ptrace.h>
#include <linux/sched.h>

struct data_t {
    u64 stacks[4];
};

BPF_HASH(gcored_samples, u32, struct data_t, 128);

void trace_stack(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid();
    struct data_t data = {};
    u64 ret = bpf_get_stack(ctx, data.stacks, sizeof(data.stacks), BPF_F_USER_STACK);
    if (ret > 0) {
        gcored_samples.update(&pid, &data);
    }
}
```
*/
// NewBPFBacktracer creates a BPF Backtracer struct
// that allocates map, prog and attaches kprobe.
func NewBPFBacktracer() (*BpfBacktracer, error) {
	samplesMap, err := ebpf.NewMap(gCoreSamlesMapSpec)
	if err != nil {
		return nil, err
	}
	// TODO: configurable
	samplesMap.Pin(filepath.Join("/sys/fs/bpf", GCoreSamlesMapName))
	samplesMap.Freeze()
	var progSpec = &ebpf.ProgramSpec{
		Name:    "gcoredumper_core_handler",
		Type:    ebpf.Kprobe,
		License: "GPL",
		Instructions: asm.Instructions{
			// r6 = r1
			asm.Mov.Reg(asm.R6, asm.R1),
			// call bpf_get_current_pid_tgid#14
			asm.FnGetCurrentPidTgid.Call(),
			// *(u32*)(r10 -4) = r0
			asm.StoreMem(asm.R10, -4, asm.R0, asm.Word),
			// r1 = 0
			asm.Mov.Imm(asm.R1, 0),
			// allocate sample struct
			// *(u64*)(r10 -16) = r1
			// *(u64*)(r10 -24) = r1
			// *(u64*)(r10 -32) = r1
			// *(u64*)(r10 -40) = r1
			asm.StoreMem(asm.RFP, -16, asm.R0, asm.DWord),
			asm.StoreMem(asm.RFP, -24, asm.R0, asm.DWord),
			asm.StoreMem(asm.RFP, -32, asm.R0, asm.DWord),
			asm.StoreMem(asm.RFP, -40, asm.R0, asm.DWord),
			// bpf_get_stack args:
			// buf*
			// r2 = r10
			asm.Mov.Reg(asm.R2, asm.R10),
			// r2 += -40
			asm.Add.Imm(asm.R2, -40),
			// ctx*
			// r1 = r6
			asm.Mov.Reg(asm.R1, asm.R6),
			// size
			// r3 = 32
			asm.Mov.Imm(asm.R3, 32),
			// flags
			// r4 = 256
			asm.Mov.Imm(asm.R4, unix.BPF_F_USER_STACK),
			// call bpf_get_stack#67
			asm.FnGetStack.Call(),
			// r0 <<= 32
			asm.LSh.Imm(asm.R0, 32),
			// r0 >>= 32
			asm.RSh.Imm(asm.R0, 32),
			// if r0 == 0 goto exit
			asm.JEq.Imm(asm.R0, 0, "exit"),
			// bpf_map_update_elem args
			// map
			// r1 = <map at fd>
			asm.LoadMapPtr(asm.R1, samplesMap.FD()),
			// key
			// r2 = r10
			asm.Mov.Reg(asm.R2, asm.R10),
			// r2 += -4
			asm.Add.Imm(asm.R2, -4),
			// value
			// r3 = r10
			asm.Mov.Reg(asm.R3, asm.R10),
			// r3 += -40
			asm.Add.Imm(asm.R3, -40),
			// flag
			// r4 = 0
			asm.Mov.Imm(asm.R4, unix.BPF_ANY),
			// call bpf_map_update_elem#2
			asm.FnMapUpdateElem.Call(),
			asm.Mov.Imm(asm.R0, 0).Sym("exit"),
			// exit
			asm.Return(),
		},
	}
	prog, err := ebpf.NewProgram(progSpec)
	if err != nil {
		samplesMap.Close()
		return nil, err
	}
	kprobe, err := link.Kprobe("do_coredump", prog)
	if err != nil {
		samplesMap.Close()
		prog.Close()
		return nil, err
	}
	return &BpfBacktracer{
		samplesMap: samplesMap,
		prog:       prog,
		link:       kprobe,
	}, nil
}

func (b *BpfBacktracer) Close() error {
	b.link.Close()
	b.prog.Close()
	b.samplesMap.Unpin()
	b.samplesMap.Close()
	return nil
}

type Key uint32

type Backtrace struct {
	Vaddrs [FramesNumber]uint64
}

func (b *Backtrace) UnmarshalBinary(buf []byte) error {
	return binary.Read(bytes.NewReader(buf), binary.LittleEndian, &b.Vaddrs)
}

var (
	_ encoding.BinaryUnmarshaler = (*Backtrace)(nil)
)

func LoadBacktracesMap() (*ebpf.Map, error) {
	return LoadBacktracesMapFromPath(filepath.Join("/sys/fs/bpf", GCoreSamlesMapName))
}

func LoadBacktracesMapFromPath(path string) (*ebpf.Map, error) {
	return ebpf.LoadPinnedMap(path, &ebpf.LoadPinOptions{
		ReadOnly: true,
	})
}
