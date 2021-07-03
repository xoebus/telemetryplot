package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	bs, err := ioutil.ReadFile("mx5.ibt")
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}

	parse(bs)
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

func parse(bs []byte) {
	headerBytes := make([]byte, 112)
	if _, err := io.ReadFull(r(bs[:112]), headerBytes); err != nil {
		log.Fatal(err)
	}
	var header TelemetryHeader
	if err := binary.Read(bytes.NewReader(headerBytes), binary.LittleEndian, &header); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", header)

	diskHeaderBytes := make([]byte, 32)
	if _, err := io.ReadFull(r(bs[112:112+32]), diskHeaderBytes); err != nil {
		log.Fatal(err)
	}
	var diskHeader DiskHeader
	if err := binary.Read(bytes.NewReader(diskHeaderBytes), binary.LittleEndian, &diskHeader); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", diskHeader)

	sessionInfo := make([]byte, header.SessionInfoLength)
	if _, err := io.ReadFull(r(bs[header.SessionInfoOffset:header.SessionInfoOffset+header.SessionInfoLength]), sessionInfo); err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("%s\n", string(sessionInfo))

	varHeaders := []VarHeader{}
	for i := 0; i < int(header.NumVars); i++ {
		start := i * 144
		end := start + 144
		var varHeader VarHeader
		if err := binary.Read(r(bs[start:end]), binary.LittleEndian, &varHeader); err != nil {
			log.Fatal(err)
		}
		varHeaders = append(varHeaders, varHeader)
	}
	for _, varHeader := range varHeaders {
		fmt.Printf("%#v\n", varHeader.Name())
		fmt.Printf("%#v\n", varHeader.Description())
		fmt.Printf("%#v\n", varHeader.Unit())
		fmt.Printf("%#v\n", varHeader.Type)
		fmt.Println()
	}

	varMap := make(map[string]VarHeader)
	for _, varHeader := range varHeaders[1:] { // TODO: why is the first record corrupted?
		varMap[strings.ToLower(varHeader.Name())] = varHeader
	}

	trace, err := os.Create("trace.dat")
	if err != nil {
		log.Fatal(err)
	}
	defer trace.Close()
	w := io.MultiWriter(trace, os.Stdout)

	count := 0
	for {
		start := header.BufOffset + (int32(count) * header.BufLen)
		end := start + header.BufLen
		if int(end) > len(bs) {
			break
		}
		buf := bs[start:end]

		lap := varMap["lap"]
		// fmt.Println(lap.Type)
		lapSize := int32(4)
		lapBuf := buf[lap.Offset : lap.Offset+lapSize]
		var lapInt int32
		if err := binary.Read(r(lapBuf), binary.LittleEndian, &lapInt); err != nil {
			log.Fatal(err)
		}

		sessionTime := varMap["sessiontime"]
		// fmt.Println(sessionTime.Type)
		doubleSize := int32(8)
		timeBuf := buf[sessionTime.Offset : sessionTime.Offset+doubleSize]
		var timeDouble float64
		if err := binary.Read(r(timeBuf), binary.LittleEndian, &timeDouble); err != nil {
			log.Fatal(err)
		}

		speed := varMap["speed"]
		// fmt.Println(speed.Type)
		size := int32(4)
		speedBuf := buf[speed.Offset : speed.Offset+size]
		var speedFloat float32
		if err := binary.Read(r(speedBuf), binary.LittleEndian, &speedFloat); err != nil {
			log.Fatal(err)
		}
		// fmt.Println("speed:", timeDouble, speedFloat)

		throttle := varMap["throttle"]
		throttleSize := int32(4)
		throttleBuf := buf[throttle.Offset : throttle.Offset+throttleSize]
		var throttleFloat float32
		if err := binary.Read(r(throttleBuf), binary.LittleEndian, &throttleFloat); err != nil {
			log.Fatal(err)
		}

		brake := varMap["brake"]
		brakeSize := int32(4)
		brakeBuf := buf[brake.Offset : brake.Offset+brakeSize]
		var brakeFloat float32
		if err := binary.Read(r(brakeBuf), binary.LittleEndian, &brakeFloat); err != nil {
			log.Fatal(err)
		}

		// if lapInt == 6 {
		fmt.Fprintln(w, timeDouble, speedFloat, throttleFloat, brakeFloat)
		// }

		count++
	}
}

func r(bs []byte) io.Reader {
	return bytes.NewReader(bs)
}
