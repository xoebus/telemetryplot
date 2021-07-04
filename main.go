package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
)

var (
	telemetryFile = flag.String("telemetry", "", "path to telemetry file")
	lap           = flag.Int("lap", 1, "lap data to extract")
)

const (
	TelemetryHeaderSize = 112
	DiskHeaderSize      = 32
	VarsHeaderSize      = 144
)

func main() {
	flag.Parse()

	if *telemetryFile == "" {
		fmt.Fprintln(os.Stderr, "-telemetry must be set")
		os.Exit(1)
	}

	f, err := os.Open(*telemetryFile)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()

	if err := parse(f, *lap); err != nil {
		log.Fatalf("failed to parse file: %v", err)
	}
}

type TelemetryHeader struct {
	Version  int32
	Status   int32
	TickRate int32

	SessionInfoUpdate int32
	SessionInfoLength int32
	SessionInfoOffset int32

	NumVars         int32
	VarHeaderOffset int32

	NumBuf    int32
	BufLen    int32
	_         [12]byte
	BufOffset int32
}

type DiskHeader struct {
	StartDate   float64
	StartTime   float64
	EndTime     float64
	LapCount    int32
	RecordCount int32
}

// 144 bytes
type VarHeader struct {
	Type        int32
	Offset      int32
	Count       int32
	CountAsTime int8
	_           [3]byte
	Stuff       [128]byte
}

func (v VarHeader) Name() string {
	return strings.ReplaceAll(string(v.Stuff[0:32]), "\x00", "")
}

func (v VarHeader) Description() string {
	return strings.ReplaceAll(string(v.Stuff[32:96]), "\x00", "")
}

func (v VarHeader) Unit() string {
	return strings.ReplaceAll(string(v.Stuff[96:128]), "\x00", "")
}

type VarDef struct {
	Type reflect.Type
}

func (v VarDef) Size() int64 {
	return int64(v.Type.Bits()) / 8
}

var varMap = map[int32]VarDef{
	2: {
		Type: reflect.TypeOf(int32(0)),
	},
	4: {
		Type: reflect.TypeOf(float32(0)),
	},
	5: {
		Type: reflect.TypeOf(float64(0)),
	},
}

func parse(r io.ReaderAt, useLap int) error {
	sr := func(offset, n int64) *io.SectionReader {
		return io.NewSectionReader(r, offset, n)
	}

	var header TelemetryHeader
	if err := binary.Read(sr(0, TelemetryHeaderSize), binary.LittleEndian, &header); err != nil {
		return err
	}

	var diskHeader DiskHeader
	if err := binary.Read(sr(0, DiskHeaderSize), binary.LittleEndian, &diskHeader); err != nil {
		return err
	}

	// TODO: Work out how to parse session info
	// sessionInfo := make([]byte, header.SessionInfoLength)
	// if _, err := io.ReadFull(r(bs[header.SessionInfoOffset:header.SessionInfoOffset+header.SessionInfoLength]), sessionInfo); err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%s\n", string(sessionInfo))

	vars := make(map[string]VarHeader, header.NumVars)
	for i := 0; i < int(header.NumVars); i++ {
		start := int64(i) * VarsHeaderSize
		var varHeader VarHeader
		if err := binary.Read(sr(start, VarsHeaderSize), binary.LittleEndian, &varHeader); err != nil {
			return err
		}
		name := strings.ToLower(varHeader.Name())
		vars[name] = varHeader
	}

	foundSamples := false
	count := 0
	for {
		start := int64(header.BufOffset) + (int64(count) * int64(header.BufLen))
		varReader := sr(start, int64(header.BufLen))

		lap, done, err := extractVar(varReader, vars, "lap")
		if err != nil {
			return err
		}
		if done {
			break
		}

		lapDist, done, err := extractVar(varReader, vars, "lapdist")
		if err != nil {
			return err
		}
		if done {
			break
		}

		speed, done, err := extractVar(varReader, vars, "speed")
		if err != nil {
			return err
		}
		if done {
			break
		}

		throttle, done, err := extractVar(varReader, vars, "throttle")
		if err != nil {
			return err
		}
		if done {
			break
		}

		brake, done, err := extractVar(varReader, vars, "brake")
		if err != nil {
			return err
		}
		if done {
			break
		}

		if lap.(int32) == int32(useLap) {
			foundSamples = true
			fmt.Println(lapDist, speed, throttle, brake)
		}

		count++
	}

	if !foundSamples {
		fmt.Println("no samples found for that lap, try a different one")
	}

	return nil
}

func extractVar(r io.ReaderAt, vars map[string]VarHeader, name string) (interface{}, bool, error) {
	lapVar, found := vars[name]
	if !found {
		return nil, false, fmt.Errorf("variable name not found: %s", name)
	}
	typ, found := varMap[lapVar.Type]
	if !found {
		return nil, false, fmt.Errorf("type not found: %d", lapVar.Type)
	}
	varReader := io.NewSectionReader(r, int64(lapVar.Offset), typ.Size())

	switch typ.Type.Kind() {
	case reflect.Int32:
		var val int32
		if err := binary.Read(varReader, binary.LittleEndian, &val); err != nil {
			if err == io.EOF {
				return nil, true, nil
			}
			return nil, false, err
		}
		return val, false, nil
	case reflect.Float32:
		var val float32
		if err := binary.Read(varReader, binary.LittleEndian, &val); err != nil {
			if err == io.EOF {
				return nil, true, nil
			}
			return nil, false, err
		}
		return val, false, nil
	case reflect.Float64:
		var val float64
		if err := binary.Read(varReader, binary.LittleEndian, &val); err != nil {
			if err == io.EOF {
				return nil, true, nil
			}
			return nil, false, err
		}
		return val, false, nil
	default:
		panic(typ.Type.Kind())
	}
}
