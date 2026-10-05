// Package jubjub calls the real Jubjub FROST verification logic — compiled
// from the same Rust (zkcrypto/jubjub-backed) code orbis-rs's signing side
// uses — via a pure-Go WASM runtime (wazero; no cgo, no Rust toolchain
// required to build or run this package). The verify logic itself lives in
// the `jubjub-wasm` crate in the separate orbis-rs repo
// (crates/jubjub-wasm, see its README for how to rebuild and update the
// vendored binary below); this package is only the Go<->WASM calling
// convention, plus the vendored compiled artifact (wasm/jubjub_wasm.wasm,
// embedded via //go:embed).
package jubjub

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed wasm/jubjub_wasm.wasm
var wasmBytes []byte

var (
	runtime     wazero.Runtime
	compiled    wazero.CompiledModule
	compileOnce sync.Once
	compileErr  error
)

func ensureCompiled(ctx context.Context) error {
	compileOnce.Do(func() {
		runtime = wazero.NewRuntime(ctx)
		compiled, compileErr = runtime.CompileModule(ctx, wasmBytes)
	})
	return compileErr
}

// moduleInstance holds one fresh wasm module instance plus the address of
// its shared static buffer. A fresh instance is created per call (see
// callExport) rather than reused across calls: this isn't a hot path for a
// chain verifying occasional threshold signatures, and it guarantees a trap
// on one call (e.g. from an unexpected internal panic) can never poison any
// other, unrelated call.
type moduleInstance struct {
	mod    api.Module
	bufPtr uint32
}

func newInstance(ctx context.Context) (*moduleInstance, error) {
	if err := ensureCompiled(ctx); err != nil {
		return nil, fmt.Errorf("jubjub: failed to compile embedded wasm module: %w", err)
	}
	mod, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("jubjub: failed to instantiate wasm module: %w", err)
	}
	bufPtrFn := mod.ExportedFunction("buf_ptr")
	results, err := bufPtrFn.Call(ctx)
	if err != nil {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("%w: buf_ptr: %w", ErrWasmTrap, err)
	}
	return &moduleInstance{mod: mod, bufPtr: uint32(results[0])}, nil
}

func (m *moduleInstance) close(ctx context.Context) {
	_ = m.mod.Close(ctx)
}

// writeInputs writes buf starting at this instance's shared buffer address.
func (m *moduleInstance) writeInputs(buf []byte) error {
	if !m.mod.Memory().Write(m.bufPtr, buf) {
		return fmt.Errorf("jubjub: wasm memory write out of range (len=%d)", len(buf))
	}
	return nil
}

// readOutput reads n bytes back from the start of the shared buffer.
func (m *moduleInstance) readOutput(n uint32) ([]byte, error) {
	out, ok := m.mod.Memory().Read(m.bufPtr, n)
	if !ok {
		return nil, fmt.Errorf("jubjub: wasm memory read out of range (len=%d)", n)
	}
	// Memory().Read returns a view into the module's live memory; copy it
	// out since the module (and its memory) may be closed shortly after.
	cp := make([]byte, n)
	copy(cp, out)
	return cp, nil
}

func (m *moduleInstance) callI32(ctx context.Context, fnName string, args ...uint64) (int32, error) {
	fn := m.mod.ExportedFunction(fnName)
	if fn == nil {
		return 0, fmt.Errorf("jubjub: wasm module has no exported function %q", fnName)
	}
	results, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("%w: %s: %w", ErrWasmTrap, fnName, err)
	}
	return int32(uint32(results[0])), nil
}

// callVerifyLikeI32 is the shared shape for verify/is_identity_pubkey/
// derive_public_key/test_sign: instantiate a fresh module, write the given
// input bytes to the shared buffer starting at offset 0, call fnName with
// the given trailing args (e.g. a message length), and return its raw i32
// result.
func callVerifyLikeI32(ctx context.Context, fnName string, input []byte, args ...uint64) (int32, error) {
	inst, err := newInstance(ctx)
	if err != nil {
		return 0, err
	}
	defer inst.close(ctx)

	if len(input) > 0 {
		if err := inst.writeInputs(input); err != nil {
			return 0, err
		}
	}
	return inst.callI32(ctx, fnName, args...)
}

// callOutputBytes is the shared shape for derive_public_key/test_sign:
// instantiate, write input, call fnName, and on success (result == expected
// output length) read that many bytes back from the start of the buffer.
func callOutputBytes(ctx context.Context, fnName string, input []byte, outLen uint32, args ...uint64) ([]byte, error) {
	inst, err := newInstance(ctx)
	if err != nil {
		return nil, err
	}
	defer inst.close(ctx)

	if err := inst.writeInputs(input); err != nil {
		return nil, err
	}
	result, err := inst.callI32(ctx, fnName, args...)
	if err != nil {
		return nil, err
	}
	if result != int32(outLen) {
		return nil, ErrInvalidScalar
	}
	return inst.readOutput(outLen)
}
