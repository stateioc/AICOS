package node_reporter

import (
	"bytes"
	"fmt"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	"github.com/shirou/gopsutil/v3/cpu"
	levenshtein "github.com/texttheater/golang-levenshtein/levenshtein"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

func getCPUChipInfo(chipCodes map[string]string) (chipType, chipModel string, chipNum int, err error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return "", "", 0, fmt.Errorf("error getting CPU info: %v", err)
	}

	// CPU型号
	chipType = "00001"

	physicalIDs := make(map[string]bool)
	for _, info := range cpuInfo {
		physicalIDs[info.PhysicalID] = true
	}

	chipNum = len(physicalIDs)
	fmt.Println("Physical cores:", chipNum)

	chipModel = extractChipProcessorName(cpuInfo[0].ModelName, chipCodes)
	return chipType, chipModel, chipNum, nil
}

func getGPUChipInfo(chipCodes map[string]string) (chipType, chipModel string, chipNum int, err error) {
	// GPU型号
	chipGPUType, chipGPUModel, chipGPUNum := getGPUNumberModelLSPCI(chipCodes)

	if (chipGPUNum) != 0 {
		chipType = chipGPUType
		chipModel = chipGPUModel
		chipModel = "00000000"
		chipNum = chipGPUNum
	}

	return chipType, chipModel, chipNum, nil
}

// 获取芯片型号
func extractChipProcessorName(modelName string, chipCodes map[string]string) string {
	var matchedModel string
	var maxSimilarity float64
	for model := range chipCodes {
		sim := similarity(strings.ToLower(modelName), strings.ToLower(model))
		if sim > maxSimilarity {
			maxSimilarity = sim
			matchedModel = model
		}
	}

	if maxSimilarity < 0.8 {
		fmt.Printf("No chip model found that matches %s with at least 80%% similarity\n", modelName)
		return "11111111"
	}

	chipCode := chipCodes[matchedModel]
	fmt.Printf("Chip model: %s, Code: %s\n", matchedModel, chipCode)

	return chipCode
}

func similarity(s1, s2 string) float64 {
	ratio := 1 - float64(levenshtein.DistanceForStrings([]rune(s1), []rune(s2), levenshtein.DefaultOptions))/float64(max(len(s1), len(s2)))
	return ratio
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getCPUChipFlops() float64 {
	cpuInfo, err := cpu.Info()
	if err != nil {
		fmt.Println("Error getting CPU info:", err)
		return 0
	}

	if len(cpuInfo) == 0 {
		fmt.Println("No CPU info found")
		return 0
	}

	firstCPU := cpuInfo[0]
	cpuMHz := firstCPU.Mhz
	hasAVX512 := false
	for _, flag := range firstCPU.Flags {
		if flag == "avx512f" {
			hasAVX512 = true
			break
		}
	}

	numCores := runtime.NumCPU()
	flopsPerCycle := 16.0
	if hasAVX512 {
		flopsPerCycle = 32.0
	}

	cpuPower := float64(numCores) * cpuMHz / 1000 * flopsPerCycle
	fmt.Printf("CPU Compute Capacity: %.2f\n", cpuPower)
	return cpuPower
}

func getGPUChipFlops() float64 {
	cpuInfo, err := cpu.Info()
	if err != nil {
		fmt.Println("Error getting CPU info:", err)
		return 0
	}

	if len(cpuInfo) == 0 {
		fmt.Println("No CPU info found")
		return 0
	}

	firstCPU := cpuInfo[0]
	cpuMHz := firstCPU.Mhz
	hasAVX512 := false
	for _, flag := range firstCPU.Flags {
		if flag == "avx512f" {
			hasAVX512 = true
			break
		}
	}

	numCores := runtime.NumCPU()
	flopsPerCycle := 16.0
	if hasAVX512 {
		flopsPerCycle = 32.0
	}

	cpuPower := float64(numCores) * cpuMHz / 1000 * flopsPerCycle
	fmt.Printf("GPU Compute Capacity: %.2f\n", cpuPower)
	return cpuPower
}

func getGPUNumberModelNVML(chipCodes map[string]string) (chipType string, chipModel string, chipNum int) {
	err := nvml.Init()
	fmt.Println("nvml.Init err: ", err)

	if err == nil {
		chipType = "00000"
		chipNum = 0

		defer nvml.Shutdown()
		n, err := nvml.GetDeviceCount()
		fmt.Println("gpu number: ", n)
		if err == nil && n > 0 {
			chipNum = int(n)
			// 获取第一个 GPU 的型号
			device, err := nvml.NewDevice(0)
			if err == nil {
				fmt.Println(*device.Model)
				chipModel = extractChipProcessorName(*device.Model, chipCodes)
			}
		} else {
			chipModel = "11111111"
		}
	}
	return chipType, chipModel, chipNum
}

func getGPUNumberModelLSPCI(chipCodes map[string]string) (chipType string, chipModel string, chipNum int) {
	chipType = "00000"
	chipModel = "11111111"
	chipNum = 0

	cmd := exec.Command("lspci")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		fmt.Println("Error running lspci:", err)
		return chipType, chipModel, chipNum
	}

	lines := strings.Split(out.String(), "\n")
	gpuRegex := regexp.MustCompile(`NVIDIA Corporation (\w+)`)
	gpuModels := make(map[string]int)

	for _, line := range lines {
		if strings.Contains(line, "3D controller") {
			matches := gpuRegex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				model := matches[1]
				gpuModels[model]++
			}
		}
	}

	maxCount := 0
	maxModel := ""
	for model, count := range gpuModels {
		if count > maxCount {
			maxCount = count
			maxModel = model
		}
	}

	fmt.Printf("%s: %d\n", maxModel, maxCount)

	chipNum = maxCount
	chipModel = extractChipProcessorName(maxModel, chipCodes)
	return chipType, chipModel, chipNum
}
