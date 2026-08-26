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
