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

	// qos
	portNumberRegex = `var portNumber = (\d+);`
	bcInfoRegex     = `bcInfo=new Array\((.*?)\)`

	// cable diagnostics
	cableStateRegex  = `var cablestate = \[(.*?)\];`
	cableLengthRegex = `var cablelength = \[(.*?)\];`
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

func parsePortSpeeds(html string, limit int) [][]int {
	regex := regexp.MustCompile(bcInfoRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([][]int, 0)
	}

	raw := matches[0][1]
	list := strings.Split(raw, ",")

	port := 0
	output := make([][]int, limit)
	for _, v := range list {
		if port >= limit {
			break
		}

		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return make([][]int, 0)
		}

		output[port] = append(output[port], i)

		if len(output[port]) == 2 {
			port++
		}
	}

	return output
}

func parseIntList(html, pattern string, limit int) []int {
	regex := regexp.MustCompile(pattern)
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

		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return []int{}
		}

		output[c] = i
	}

	return output[:limit]
}

func parseCableState(html string, limit int) []int {
	return parseIntList(html, cableStateRegex, limit)
}

func parseCableLength(html string, limit int) []int {
	return parseIntList(html, cableLengthRegex, limit)
}

func cableStateLabel(n int) string {
	switch n {
	case 0:
		return "No Cable"
	case 1:
		return "Normal"
	case 2:
		return "Open"
	case 3:
		return "Short"
	case 4:
		return "Open & Short"
	case 5:
		return "Cross Cable"
	case -1:
		return "Untested"
	default:
		return "Others"
	}
}
