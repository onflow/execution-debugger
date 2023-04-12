package debugger

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type LogInterceptor struct {
	ComputationIntensities map[uint64]uint64 `json:"computationIntensities"`
	MemoryIntensities      map[uint64]uint64 `json:"memoryIntensities"`

	log      zerolog.Logger
	filename string
}

func NewLogInterceptor(log zerolog.Logger, directory string) *LogInterceptor {
	return &LogInterceptor{
		ComputationIntensities: map[uint64]uint64{},
		MemoryIntensities:      map[uint64]uint64{},
		log:                    log,
		filename:               directory + "/computation_intensities.csv",
	}
}

var _ io.Writer = &LogInterceptor{}

type computationIntensitiesLog struct {
	ComputationIntensities map[uint64]uint64 `json:"computationIntensities"`
	MemoryIntensities      map[uint64]uint64 `json:"memoryIntensities"`
}

func (l *LogInterceptor) Write(p []byte) (n int, err error) {
	fmt.Println("#log", string(p))

	if strings.Contains(string(p), "computationIntensities") {
		var log computationIntensitiesLog
		err := json.Unmarshal(p, &log)
		if err != nil {
			return 0, err
		}
		l.ComputationIntensities = log.ComputationIntensities
		l.MemoryIntensities = log.MemoryIntensities

		return len(p), nil
	}
	return len(p), nil
}

func (l *LogInterceptor) Close() error {
	err := os.MkdirAll(filepath.Dir(l.filename), os.ModePerm)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(l.filename)
	if err != nil {
		return err
	}
	defer func(csvFile *os.File) {
		err := csvFile.Close()
		if err != nil {
			l.log.Warn().
				Err(err).
				Msg("Could not close csv file.")
		}
	}(csvFile)

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	err = writer.Write([]string{"*Computation Kind", "Intensity"})
	if err != nil {
		return err
	}
	for i, q := range l.ComputationIntensities {
		key, ok := computationKindNameMap[i]
		if !ok {
			key = strconv.Itoa(int(i))
		}

		err := writer.Write([]string{key, strconv.Itoa(int(q))})
		if err != nil {
			return err
		}
	}

	return nil
}

var computationKindNameMap = map[uint64]string{
	1001: "*Statement",
	1002: "*Loop",
	1003: "*FunctionInvocation",
	1010: "CreateCompositeValue",
	1011: "TransferCompositeValue",
	1012: "DestroyCompositeValue",
	1025: "CreateArrayValue",
	1026: "TransferArrayValue",
	1027: "DestroyArrayValue",
	1040: "CreateDictionaryValue",
	1041: "TransferDictionaryValue",
	1042: "DestroyDictionaryValue",
	1100: "STDLIBPanic",
	1101: "STDLIBAssert",
	1102: "STDLIBUnsafeRandom",
	1108: "STDLIBRLPDecodeString",
	1109: "STDLIBRLPDecodeList",
	2001: "Hash",
	2002: "VerifySignature",
	2003: "AddAccountKey",
	2004: "AddEncodedAccountKey",
	2005: "AllocateStorageIndex",
	2006: "*CreateAccount",
	2007: "EmitEvent",
	2008: "GenerateUUID",
	2009: "GetAccountAvailableBalance",
	2010: "GetAccountBalance",
	2011: "GetAccountContractCode",
	2012: "GetAccountContractNames",
	2013: "GetAccountKey",
	2014: "GetBlockAtHeight",
	2015: "GetCode",
	2016: "GetCurrentBlockHeight",
	2017: "GetProgram",
	2018: "GetStorageCapacity",
	2019: "GetStorageUsed",
	2020: "*GetValue",
	2021: "RemoveAccountContractCode",
	2022: "ResolveLocation",
	2023: "RevokeAccountKey",
	2034: "RevokeEncodedAccountKey",
	2025: "SetProgram",
	2026: "*SetValue",
	2027: "UpdateAccountContractCode",
	2028: "ValidatePublicKey",
	2029: "ValueExists",
}
