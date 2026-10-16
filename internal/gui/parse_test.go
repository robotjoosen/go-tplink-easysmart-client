package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const cableDiagHTML = `<script type="text/javascript">
var maxPort = 16;
var cablestate = [1, 2, 1, 2, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 1, 2, -1];
var cablelength = [6, 18, 8, 21, 8, 33, 3, 26, 0, 0, 0, 0, 0, 0, 1, 1, -1];
</script>`

func Test_parseCableState(t *testing.T) {
	result := parseCableState(cableDiagHTML, 16)

	assert.Equal(t, []int{1, 2, 1, 2, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 1, 2}, result)
}

func Test_parseCableLength(t *testing.T) {
	result := parseCableLength(cableDiagHTML, 16)

	assert.Equal(t, []int{6, 18, 8, 21, 8, 33, 3, 26, 0, 0, 0, 0, 0, 0, 1, 1}, result)
}

func Test_cableStateLabel(t *testing.T) {
	cases := map[int]string{
		0:  "No Cable",
		1:  "Normal",
		2:  "Open",
		3:  "Short",
		4:  "Open & Short",
		5:  "Cross Cable",
		-1: "Untested",
		42: "Others",
	}

	for in, want := range cases {
		assert.Equal(t, want, cableStateLabel(in))
	}
}

const portPageHTML = `max_port_num = 2;
state:[1,0],
link_status:[1,0],
link_info=new Array("Link Down","100M Full"),
pkts:[1,2,3,4,5,6,7,8]`

func Test_parseMaxPorts(t *testing.T) {
	assert.Equal(t, 2, parseMaxPorts(portPageHTML))
	assert.Equal(t, 0, parseMaxPorts("<html>login</html>"))
	assert.Equal(t, 0, parseMaxPorts("max_port_num = x;"))
}

func Test_parseIntList(t *testing.T) {
	cases := map[string]struct {
		html  string
		limit int
		want  []int
	}{
		"state":         {portPageHTML, 2, []int{1, 0}},
		"longer list":   {portPageHTML, 1, []int{1}},
		"short list":    {portPageHTML, 3, []int{}},
		"not a number":  {"state:[1,x]", 2, []int{}},
		"missing field": {"nothing here", 2, []int{}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseIntList(tc.html, stateRegex, tc.limit))
		})
	}
}

func Test_parsePackets(t *testing.T) {
	assert.Equal(t, [][]int{{1, 2, 3, 4}, {5, 6, 7, 8}}, parsePackets(portPageHTML, 2))
	assert.Equal(t, [][]int{{1, 2, 3, 4}}, parsePackets(portPageHTML, 1))
	assert.Empty(t, parsePackets(portPageHTML, 3))
	assert.Empty(t, parsePackets("nothing here", 2))
}

func Test_parseLinkLabels(t *testing.T) {
	assert.Equal(t, []string{"Link Down", "100M Full"}, parseLinkLabels(portPageHTML, 2))
	assert.Equal(t, []string{"Link Down", "100M Full"}, parseLinkLabels(portPageHTML, 1))
	assert.Empty(t, parseLinkLabels("nothing here", 2))
}

func Test_parsePortSpeeds(t *testing.T) {
	html := "var portNumber = 2;\nvar bcInfo = new Array(\n100,200,0,\n300,400,1,\n0);"

	assert.Equal(t, [][]int{{100, 200, 0}, {300, 400, 1}}, parsePortSpeeds(html, 2))
	assert.Equal(t, [][]int{{100, 200, 0}}, parsePortSpeeds(html, 1))
	assert.Empty(t, parsePortSpeeds(html, 4))
	assert.Empty(t, parsePortSpeeds("bcInfo = new Array(100, x, 0)", 1))
}

func Test_parsePortNumber(t *testing.T) {
	assert.Equal(t, 16, parsePortNumber("var portNumber = 16;"))
	assert.Equal(t, 0, parsePortNumber("nothing here"))
}

func Test_parseDeviceInfo(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, parseDeviceInfo("\"a\"\n\"b\"\n"))
	assert.Empty(t, parseDeviceInfo("no quotes"))
}
