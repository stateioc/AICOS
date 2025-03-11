package node_reporter

import (
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
)

func getMaxConfiguredBandwidth() (int, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return 0, fmt.Errorf("error getting network interfaces: %v", err)
	}

	maxBandwidth := 0
	for _, link := range links {
		// 跳过虚拟接口
		if strings.HasPrefix(link.Attrs().Name, "lo") || strings.HasPrefix(link.Attrs().Name, "docker") || strings.HasPrefix(link.Attrs().Name, "virbr") {
			continue
		}

		// 获取网络接口的速率（配置带宽）
		speed, err := getInterfaceSpeed(link.Attrs().Name)
		if err != nil {
			fmt.Printf("Error getting speed for %s: %v\n", link.Attrs().Name, err)
			continue
		}

		// 更新最大带宽
		if speed > maxBandwidth {
			maxBandwidth = speed
		}
	}

	return maxBandwidth, nil
}

func getInterfaceSpeed(interfaceName string) (int, error) {
	cmd := exec.Command("ethtool", interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Speed:") {
			re := regexp.MustCompile(`\d+`)
			speedStr := re.FindString(line)
			speed, err := strconv.Atoi(speedStr)
			if err != nil {
				return 0, err
			}
			return speed, nil
		}
	}

	return 0, fmt.Errorf("Speed not found for interface %s", interfaceName)
}

func ipv4ToBinary(ips string) string {
	ip := net.ParseIP(ips)
	if ip == nil {
		fmt.Println("Invalid IP address")
		return fmt.Sprintf("%032b", 0)
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		fmt.Println("Not an IPv4 address")
		return fmt.Sprintf("%032b", 0)
	}

	// Convert the 4-byte IPv4 address to a 32-bit unsigned integer
	ipv4Int := binary.BigEndian.Uint32(ipv4)

	// Convert the integer to a binary string
	ipv4String := fmt.Sprintf("%032b", ipv4Int)
	//fmt.Println(ipv4String)

	return ipv4String
}
