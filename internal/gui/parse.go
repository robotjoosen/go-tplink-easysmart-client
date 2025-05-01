package gui

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	// monitoring
	stateRegex           = `state:\[(.*)\]`
	linkStatusRegex      = `link_status:\[(.*)\]`
	linkStatusLabelRegex = `link_info=new Array\(([a-zA-Z" ,0-9]*)\)`
	packetsRegex         = `pkts:\[(.*)\]`
	maxPortsRegex        = `max_port_num = (.*);`

	// device info
	deviceInfoRegex = `".*"`
)

func parsePackets(html string, limit int) [][]int {
	regex := regexp.MustCompile(packetsRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([][]int, 0)
	}

	raw := matches[0][1]
	list := strings.Split(raw, ",")

	port := 0
	output := make([][]int, limit)
	for c, v := range list {
		if port >= limit {
			break
		}

		i, err := strconv.Atoi(v)
		if err != nil {
			return make([][]int, 0)
		}

		output[port] = append(output[port], i)

		if c%4 == 3 {
			port++
		}
	}

	return output
}

func parseMaxPorts(html string) int {
	regex := regexp.MustCompile(maxPortsRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return 0
	}

	raw := matches[0][1]
	i, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return i
}

func parseStatusNumber(html string, limit int) []int {
	regex := regexp.MustCompile(stateRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([]int, 0)
	}

	raw := matches[0][1]
	list := strings.Split(raw, ",")

	output := make([]int, limit)
	for c, v := range list {
		if c >= limit {
			break
		}

		i, err := strconv.Atoi(v)
		if err != nil {
			return []int{}
		}

		output[c] = i
	}

	return output[:limit]
}

func parseLinkStatus(html string, limit int) []int {
	regex := regexp.MustCompile(linkStatusRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([]int, 0)
	}

	raw := matches[0][1]
	list := strings.Split(raw, ",")

	output := make([]int, limit)
	for c, v := range list {
		if c >= limit {
			break
		}

		i, err := strconv.Atoi(v)
		if err != nil {
			return []int{}
		}

		output[c] = i
	}

	return output[:limit]
}

func parseLinkLabels(html string, limit int) []string {
	regex := regexp.MustCompile(linkStatusLabelRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([]string, 0)
	}

	raw := matches[0][1]
	raw = strings.Replace(raw, `"`, ``, -1)

	return strings.SplitN(raw, ",", limit)
}

func parseDeviceInfo(html string) []string {
	regex := regexp.MustCompile(deviceInfoRegex)
	matches := regex.FindAllStringSubmatch(html, 7)
	if len(matches) == 0 {
		return make([]string, 0)
	}

	output := make([]string, 0)
	for _, m := range matches {
		output = append(output, strings.Replace(m[0], `"`, ``, -1))
	}

	return output
}
